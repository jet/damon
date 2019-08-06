// +build windows

package win32

import (
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
	ExitTimeout time.Duration
	osProcess   *os.Process
	mu          sync.RWMutex
	suspended   bool
	doneCh      chan struct{}

	resultLock sync.Mutex
	result     atomic.Value
}

// ProcessResult is the result of the process after it completed
type ProcessResult struct {
	Err        error
	ExitStatus int
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

type AffinityMask uint32

// AffinityMask returns the process affinity mask and system affinity mask
func (p *Process) AffinityMask() (AffinityMask, AffinityMask, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	phProc, err := openProcess(_PROCESS_QUERY_INFORMATION, false, p.Pid())
	if err != nil {
		return 0, 0, nil
	}
	defer CloseHandleLogErr(*phProc, "win32: failed to close process handle")
	pam, sam, err := getProcessAffinityMask(*phProc)
	return AffinityMask(pam), AffinityMask(sam), err
}

// MemoryInfo gets the process memory information
//
// There is probably a different way to get more granular memory metrics,
// but we're avoiding WMI because in practice it had a tendency of blocking and crashing if the queries per second is too great
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

// Wait until the process exits and return the results.
// exitCh is used to signal the process to exit early
// returns an error if the process was not started
func (p *Process) Wait(exitCh <-chan struct{}) (*ProcessResult, error) {
	// fast atomic check for existing result
	if v := p.result.Load(); v != nil {
		return v.(*ProcessResult), nil
	}
	p.resultLock.Lock()
	defer p.resultLock.Unlock()
	// there may have been another Wait that exited after this Lock
	// so check again before doing the whole thing
	if v := p.result.Load(); v != nil {
		return v.(*ProcessResult), nil
	}
	type waitStatus struct {
		State *os.ProcessState
		Err   error
	}
	resultCh := make(chan waitStatus, 1)
	if exitCh == nil {
		exitCh = make(chan struct{}) // never exit
	}
	go func() {
		select {
		case <-exitCh:
			Logf("win32: command termination requested")
			// received a request to exit the process
		case <-p.doneCh:
			Logf("win32: command completed")
			// done before exit signal received
			return
		}
		// try to exit gracefully
		if err := generateConsoleCtrlEvent(syscall.CTRL_BREAK_EVENT, p.Pid()); err != nil {
			// ctrl+break not sent, kill now
			LogError(p.osProcess.Kill(), "win32: could not kill process")
			return
		}
		select {
		case <-p.doneCh:
			Logf("win32: command completed")
			return
		case <-time.After(p.ExitTimeout):
			// give up -- send kill signal
			LogError(p.osProcess.Kill(), "win32: could not kill process")
		}
	}()
	go func() {
		defer close(p.doneCh)
		Logf("win32: Cmd.Wait")
		state, err := p.osProcess.Wait()
		Logf("win32: Cmd.Wait complete")
		LogError(err, "win32: Cmd.Wait error")
		resultCh <- waitStatus{State: state, Err: err}
	}()
	res := <-resultCh
	Logf("win32: process completed")
	pr := &ProcessResult{
		Err:        res.Err,
		ExitStatus: getExitCode(res.State, res.Err),
	}
	p.result.Store(pr)
	return pr, nil
}

func getExitCode(state *os.ProcessState, err error) int {
	if state != nil && state.Exited() {
		return state.ExitCode()
	}
	if err != nil {
		return ExitStatusError
	}
	return ExitStatusUnknown
}

// Pid returns the process ID
func (p *Process) Pid() uint32 {
	if proc := p.osProcess; proc != nil {
		return uint32(proc.Pid)
	}
	return 0
}

// Resume will resume the process created with Suspend
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

// Try to gracefully shut down the process before killing
func (p *Process) Shutdown(timeout time.Duration) error {
	pr := p.result.Load()
	if pr != nil {
		return nil
	}
	// try to exit gracefully
	if err := generateConsoleCtrlEvent(syscall.CTRL_BREAK_EVENT, p.Pid()); err != nil {
		// ctrl+break not sent, kill now
		LogError(p.osProcess.Kill(), "win32: could not kill process")
	}
	select {
	case <-p.doneCh:
		return nil
	case <-time.After(timeout):
		return p.Kill()
	}
}

// Kill the running process
func (p *Process) Kill() error {
	pr := p.result.Load()
	if pr != nil {
		return nil
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.osProcess.Kill()
}

func AccessToken(token *Token) func(*exec.Cmd, *Process) {
	return func(command *exec.Cmd, proc *Process) {
		if command.SysProcAttr == nil {
			command.SysProcAttr = &syscall.SysProcAttr{
				CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
			}
		} else {
			command.SysProcAttr.CreationFlags |= syscall.CREATE_NEW_PROCESS_GROUP
		}
		if token != nil {
			command.SysProcAttr.Token = token.hToken
		}
	}
}

func Suspended(command *exec.Cmd, proc *Process) {
	if command.SysProcAttr == nil {
		command.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: _CREATE_SUSPENDED,
		}
	} else {
		command.SysProcAttr.CreationFlags |= _CREATE_SUSPENDED
	}
	proc.suspended = true
}

// Sets up a command with additional options
func StartProcess(command *exec.Cmd, opts ...func(cmd *exec.Cmd, proc *Process)) (*Process, error) {
	var proc Process
	for _, opt := range opts {
		opt(command, &proc)
	}
	if err := command.Start(); err != nil {
		return nil, err
	}
	proc.osProcess = command.Process
	proc.doneCh = make(chan struct{})
	return &proc, nil
}
