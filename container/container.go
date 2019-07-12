package container

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/jet/damon/log"
	"github.com/jet/damon/win32"
	"github.com/pkg/errors"
)

type Config struct {
	// EnforceCPU if set to true will enable kernel max-cpu rate enforcement
	EnforceCPU bool
	// EnforceMemory if set to true will enable memory quota
	EnforceMemory bool
	// RestrictedToken will run the process with restricted privileges
	RestrictedToken bool
	// MemoryMBLimit is the maximum committed memory that the container will allow.
	// Going over this limit will cause the program to crash with a memory allocation error.
	MemoryMBLimit int
	// CPUMHzLimit is the cpu time constraint that when fully enforced
	CPUMHzLimit int
	// CPUHardCap enforces a hard cap on the CPU time this process can get
	// If set to false, then it uses a weight
	CPUHardCap bool
}

const MBToBytes uint64 = 1024 * 1024
const MinimumCPUMHz = 100

type Container struct {
	Name string
	Config
	Logger      log.Logger
	Command     *exec.Cmd
	OnStats     OnStatsFn
	OnViolation OnViolationFn
	exitCh      <-chan struct{}
	doneCh      <-chan struct{}
	job         *win32.JobObject
	proc        *win32.Process
}

type Result struct {
	Start    time.Time
	End      time.Time
	ExitCode int
}

type LimitViolation struct {
	Type    string
	Message string
}

const (
	CPULimitViolation    = "CPU"
	MemoryLimitViolation = "Memory"
	IOLimitViolation     = "IO"
)

type ProcessStats struct {
	CPUStats
	MemoryStats
	IOStats
}

type MemoryStats struct {
	WorkingSetSizeBytes uint64
	PrivateUsageBytes   uint64
	PageFaultCount      uint64
}

type CPUStats struct {
	TotalRunTime    time.Duration
	TotalCPUTime    time.Duration
	TotalKernelTime time.Duration
	TotalUserTime   time.Duration
}

type IOStats struct {
	TotalIOOperations      uint64
	TotalReadIOOperations  uint64
	TotalWriteIOOperations uint64
	TotalOtherIOOperations uint64
	TotalTxCountBytes      uint64
	TotalTxReadBytes       uint64
	TotalTxWrittenBytes    uint64
	TotalTxOtherBytes      uint64
}

type OnStatsFn func(s ProcessStats)
type OnViolationFn func(v LimitViolation)

func (c *Container) Start() error {
	job, err := win32.CreateJobObject(c.Name)
	if err != nil {
		return errors.Wrapf(err, "unable to get create win32.JobObject")
	}
	c.job = job
	token, err := win32.CurrentProcessToken()
	if err != nil {
		return errors.Wrapf(err, "unable to get current process token")
	}
	if c.Config.RestrictedToken {
		c.Logger.Logln("creating restricted token")
		rt, err := token.CreateRestrictedToken(win32.TokenRestrictions{
			DisableMaxPrivilege: true,
			LUAToken:            true,
			DisableSIDs: []string{
				"BUILTIN\\Administrator",
			},
		})
		c.closeLogError(token, "couldn't closed process token")
		if err != nil {
			return errors.Wrapf(err, "unable to create restricted token")
		}
		token = rt
	}
	defer c.closeLogError(token, "couldn't closed process token")

	// Link up standard in/out
	c.Command.Stderr = os.Stderr
	c.Command.Stdout = os.Stdout
	c.Command.Stdin = os.Stdin

	proc, err := win32.CreateProcessWithToken(c.Command, token)
	if err != nil {
		return errors.Wrapf(err, "unable to get create process")
	}
	c.proc = proc
	if err = c.proc.StartSuspended(); err != nil {
		return err
	}
	if err = job.Assign(proc); err != nil {
		c.Logger.Error(proc.Kill(), "unable to kill child process")
		return err
	}
	eli := &win32.ExtendedLimitInformation{
		KillOnJobClose: true,
	}
	if c.Config.EnforceMemory {
		eli.JobMemoryLimit = MBToBytes * uint64(c.Config.MemoryMBLimit)
	}
	if err = c.killOnError(job.SetInformation(eli)); err != nil {
		c.closeLogError(job, "failed to close JobObject")
		return errors.Wrapf(err, "container: Could not set basic limit information")
	}
	if c.Config.EnforceCPU {
		if c.Config.CPUMHzLimit < MinimumCPUMHz {
			return errors.Errorf("CPUMHzLimit is too low. Minimum is %d", MinimumCPUMHz)
		}
		nli := &win32.NotificationLimitInformation{
			CPURateLimit: &win32.NotificationRateLimitTolerance{
				Level:    win32.ToleranceLow,
				Interval: win32.ToleranceIntervalLong,
			},
		}
		crci := &win32.CPURateControlInformation{
			Rate: &win32.CPUMaxRateInformation{
				HardCap: true,
				Rate:    win32.MHzToCPURate(uint64(c.Config.CPUMHzLimit)),
			},
			Notify: true,
		}
		if err = c.killOnError(job.SetInformation(nli)); err != nil {
			c.closeLogError(job, "failed to close JobObject")
			return errors.Wrapf(err, "container: Could not set cpu notification limits")
		}
		if err = c.killOnError(job.SetInformation(crci)); err != nil {
			c.closeLogError(job, "failed to close JobObject")
			return errors.Wrapf(err, "container: Could not set cpu rate limits")
		}
	}
	if err = c.killOnError(proc.Resume()); err != nil {
		c.closeLogError(job, "failed to close JobObject")
		return errors.Wrapf(err, "container: Could not resume process main thread")
	}
	c.exitCh = make(chan struct{})
	c.doneCh = make(chan struct{})
	if c.OnStats != nil {
		go c.pollStats()
	}
	go c.pollNotifications()
	return nil
}

func (c *Container) pollNotifications() {
	for {
		select {
		case <-c.exitCh:
			return
		case <-c.doneCh:
			return
		default:
		}
		info, err := c.job.PollNotifications()
		if err != nil {
			c.Logger.Error(err, "container: poll notifications error")
			continue
		}
		if info.Code == win32.JobObjectMsgNotificationLimit { // Limit violation
			var violations []LimitViolation
			if vi := info.LimitViolationInfo; vi != nil {
				if vi.CPURateViolation != nil {
					tolerance := ""
					switch vi.CPURateViolation.Limit {
					case 1:
						tolerance = " > 20% of the time"
					case 2:
						tolerance = " > 40% of the time"
					case 3:
						tolerance = " > 60% of the time"
					}
					violations = append(violations, LimitViolation{
						Type:    CPULimitViolation,
						Message: fmt.Sprintf("CPU Rate exceeded threshold%s", tolerance),
					})
				}
				if vi.IORateViolation != nil {
					violations = append(violations, LimitViolation{
						Type:    IOLimitViolation,
						Message: fmt.Sprintf("IO Rate exceeded threshold: %d > %d", vi.IORateViolation.Measured, vi.IORateViolation.Limit),
					})
				}
				if vi.HighMemoryViolation != nil {
					violations = append(violations, LimitViolation{
						Type:    MemoryLimitViolation,
						Message: fmt.Sprintf("Memory exceeded threshold: %d > %d", vi.HighMemoryViolation.Measured, vi.HighMemoryViolation.Limit),
					})
				}
			}
			if c.OnViolation != nil {
				for _, v := range violations {
					c.OnViolation(v)
				}
			}
		}
	}
}

func (c *Container) pollStats() {
	for {
		select {
		case <-c.exitCh:
			return
		case <-c.doneCh:
			return
		case <-time.After(10 * time.Second):
			info := &win32.JobObjectBasicAndIOAccounting{}
			if err := c.job.GetInformation(info); err != nil {
				c.Logger.Error(err, "container: get JobObjectBasicAndIOAccounting error")
				continue
			}
			meminfo, err := c.proc.MemoryInfo()
			if err != nil {
				c.Logger.Error(err, "container: get proc.MemoryInfo error")
				continue
			}
			procTime := time.Since(c.proc.StartTime())
			stats := ProcessStats{
				CPUStats: CPUStats{
					TotalRunTime:    procTime,
					TotalCPUTime:    procTime * time.Duration(runtime.NumCPU()),
					TotalKernelTime: info.Basic.TotalKernelTime,
					TotalUserTime:   info.Basic.TotalUserTime,
				},
				MemoryStats: MemoryStats{
					WorkingSetSizeBytes: meminfo.WorkingSetSize,
					PrivateUsageBytes:   meminfo.PrivateUsage,
					PageFaultCount:      uint64(meminfo.PageFaultCount),
				},
				IOStats: IOStats{
					TotalIOOperations:      info.IO.OtherOperationCount + info.IO.ReadOperationCount + info.IO.WriteOperationCount,
					TotalOtherIOOperations: info.IO.OtherOperationCount,
					TotalReadIOOperations:  info.IO.ReadOperationCount,
					TotalWriteIOOperations: info.IO.WriteOperationCount,
					TotalTxReadBytes:       info.IO.ReadTransferCount,
					TotalTxWrittenBytes:    info.IO.WriteTransferCount,
					TotalTxOtherBytes:      info.IO.OtherTransferCount,
					TotalTxCountBytes:      info.IO.ReadTransferCount + info.IO.WriteTransferCount + info.IO.OtherTransferCount,
				},
			}
			if c.OnStats != nil {
				c.OnStats(stats)
			}
		}
	}
}

func (c *Container) Wait(exitCh <-chan struct{}) (Result, error) {
	pr, err := c.proc.Wait(exitCh)
	c.Logger.Logf("process exited: %d", pr.ExitStatus)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Start:    pr.StartTime,
		End:      pr.EndTime,
		ExitCode: pr.ExitStatus,
	}, pr.Err
}

func (c *Container) killOnError(err error) error {
	if err != nil {
		c.Logger.Error(c.proc.Kill(), "unable to kill child process")
	}
	return err
}

func (c *Container) closeLogError(o io.Closer, msg string) {
	if err := o.Close(); err != nil {
		c.Logger.Error(err, msg)
	}
}
