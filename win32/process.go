// +build windows

package win32

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

const DefaultExitTimeout = time.Second * 30

// ErrProcessNotStarted is returned when an operation is performed
// on a process before it has been started.
var ErrProcessNotStarted = errors.New("process not started")

const (
	ExitStatusStartError = 253
	ExitStatusError      = 254
	ExitStatusUnknown    = 255
)

// Process wraps exec.Cmd to provide some helper functions
type Process struct {
	Cmd         *exec.Cmd
	ExitTimeout time.Duration
	Token       *Token
	mu          sync.RWMutex
	suspended   bool
	started     bool
	ended       bool
	startTime   time.Time
	endTime     time.Time
}

// ProcessResult is the result of the process after it completed
type ProcessResult struct {
	Err        error
	ExitStatus int
	StartTime  time.Time
	EndTime    time.Time
}

type ProcessMemoryInfo struct {
	PageFaultCount             uint32
	PeakWorkingSetSize         uint64
	WorkingSetSize             uint64
	QuotaPeakPagedPoolUsage    uint64
	QuotaPagedPoolUsage        uint64
	QuotaPeakNonPagedPoolUsage uint64
	QuotaNonPagedPoolUsage     uint64
	PagefileUsage              uint64
	PeakPagefileUsage          uint64
	PrivateUsage               uint64
}

// Start running the process command
// Use Wait to block until the process completes
func (p *Process) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.start()
}

// RunningDuration returns the duration that the process has been running
func (p *Process) RunningDuration() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.ended {
		return p.endTime.Sub(p.startTime)
	}
	return time.Since(p.startTime)
}

// StartTime returns the start time of the process
func (p *Process) StartTime() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.startTime
}

type AffinityMask uint32

// AffinityMask returns the process affinity mask and system affinity mask
func (p *Process) AffinityMask() (AffinityMask, AffinityMask, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.started {
		phProc, err := openProcess(_PROCESS_QUERY_INFORMATION, false, p.Pid())
		if err != nil {
			return 0, 0, nil
		}
		defer CloseHandleLogErr(*phProc, "win32: failed to close process handle")
		pam, sam, err := getProcessAffinityMask(*phProc)
		return AffinityMask(pam), AffinityMask(sam), err
	}
	return 0, 0, ErrProcessNotStarted
}

func (p *Process) MemoryInfo() (ProcessMemoryInfo, error) {
	phProc, err := openProcess(_PROCESS_QUERY_INFORMATION|_PROCESS_VM_READ, false, p.Pid())
	if err != nil {
		return ProcessMemoryInfo{}, err
	}
	defer CloseHandleLogErr(*phProc, "win32: failed to close process handle")
	minfo, err := getProcessMemoryInfo(*phProc)
	if err != nil {
		return ProcessMemoryInfo{}, err
	}
	return ProcessMemoryInfo{
		PageFaultCount:             minfo.PageFaultCount,
		PeakWorkingSetSize:         uint64(minfo.PeakWorkingSetSize),
		WorkingSetSize:             uint64(minfo.WorkingSetSize),
		QuotaPeakPagedPoolUsage:    uint64(minfo.QuotaPeakPagedPoolUsage),
		QuotaPagedPoolUsage:        uint64(minfo.QuotaPagedPoolUsage),
		QuotaPeakNonPagedPoolUsage: uint64(minfo.QuotaPeakNonPagedPoolUsage),
		QuotaNonPagedPoolUsage:     uint64(minfo.QuotaNonPagedPoolUsage),
		PagefileUsage:              uint64(minfo.PagefileUsage),
		PeakPagefileUsage:          uint64(minfo.PeakPagefileUsage),
		PrivateUsage:               uint64(minfo.PrivateUsage),
	}, nil
}

// StartSuspended starts the process with the main thread suspended
// which is useful for creating a process that should be assigned
// to a JobObject before running
// Use Process.Resume to resume the suspended process.
func (p *Process) StartSuspended() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.suspended = true
	p.Cmd.SysProcAttr.CreationFlags |= _CREATE_SUSPENDED
	return p.start()
}

func (p *Process) start() error {
	if err := p.Cmd.Start(); err != nil {
		return err
	}
	p.startTime = time.Now()
	p.started = true
	return nil
}

// Wait until the process exits and return the results.
// exitCh is used to signal the process to exit early
// returns an error if the process was not started
func (p *Process) Wait(exitCh <-chan struct{}) (*ProcessResult, error) {
	p.mu.RLock()
	if !p.started {
		p.mu.RUnlock()
		return nil, ErrProcessNotStarted
	}
	p.mu.RUnlock()
	var werr atomic.Value
	doneCh := make(chan struct{})
	if p.Cmd.Process == nil {
		return nil, fmt.Errorf("Process.Wait: process not found. Is it started?")
	}
	if exitCh == nil {
		exitCh = make(chan struct{}) // never exit
	}
	go func() {
		select {
		case <-exitCh:
			Logf("win32: command termination requested")
			// received a request to exit the process
		case <-doneCh:
			Logf("win32: command completed")
			// done before exit signal received
			return
		}
		// try to exit gracefully
		if err := generateConsoleCtrlEvent(syscall.CTRL_BREAK_EVENT, p.Pid()); err != nil {
			// ctrl+break not sent, kill now
			LogError(p.Cmd.Process.Kill(), "win32: could not kill process")
			return
		}
		select {
		case <-doneCh:
			Logf("win32: command completed")
			return
		case <-time.After(p.ExitTimeout):
			// give up -- send kill signal
			LogError(p.Cmd.Process.Kill(), "win32: could not kill process")
		}
	}()
	go func() {
		defer close(doneCh)
		Logf("win32: Cmd.Wait")
		err := p.Cmd.Wait()
		Logf("win32: Cmd.Wait complete")
		LogError(err, "win32: Cmd.Wait error")
		p.mu.Lock()
		p.ended = true
		p.endTime = time.Now()
		if err != nil {
			werr.Store(err)
		}
		p.mu.Unlock()
	}()
	<-doneCh
	Logf("win32: process completed")
	res := &ProcessResult{
		StartTime: p.startTime,
		EndTime:   p.endTime,
	}
	if e, ok := werr.Load().(error); ok {
		res.Err = e
	}
	res.ExitStatus = getExitCode(p.Cmd.ProcessState, res.Err)
	return res, nil
}

func getExitCode(state *os.ProcessState, err error) int {
	if state == nil {
		return ExitStatusUnknown
	}
	if !state.Exited() {
		return ExitStatusUnknown
	}
	if state.Success() {
		return 0
	}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
				return ws.ExitStatus()
			}
		}
		return ExitStatusError
	}
	return ExitStatusUnknown
}

// Pid returns the process ID
func (p *Process) Pid() uint32 {
	if proc := p.Cmd.Process; proc != nil {
		return uint32(proc.Pid)
	}
	return 0
}

// Resume will resume the process created with suspend=true
func (p *Process) Resume() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.suspended {
		phThread, err := openProcessMainThreadForResume(p.Pid())
		if err != nil {
			return err
		}
		defer CloseHandleLogErr(*phThread, "win32: failed to close thread handle")
		if err = resumeThread(*phThread); err != nil {
			return err
		}
		p.suspended = false
	}
	return nil
}

// Kill the running process
func (p *Process) Kill() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Cmd.Process.Kill()
}

// CreateProcessWithToken creates a process with the given access token
// which is used to limit the access rights of the command
func CreateProcessWithToken(command *exec.Cmd, token *Token) (*Process, error) {
	cmd := &Process{
		Cmd:         command,
		ExitTimeout: DefaultExitTimeout,
	}
	if command.SysProcAttr == nil {
		command.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		}
	} else {
		command.SysProcAttr.CreationFlags |= syscall.CREATE_NEW_PROCESS_GROUP
	}
	if token != nil {
		command.SysProcAttr.Token = token.hToken
		cmd.Token = token
	}
	return cmd, nil
}
