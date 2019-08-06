package container

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/jet/damon/win32"
	"github.com/pkg/errors"
)

type Config struct {
	// Name of the container
	Name string
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
	// Logger to be used for debug logs
	Logger Logger
}

const MBToBytes uint64 = 1024 * 1024
const MinimumCPUMHz = 100

type Container struct {
	Name      string
	StartTime time.Time
	Logger    Logger
	exitCh    <-chan struct{}
	doneCh    <-chan struct{}
	job       *win32.JobObject
	proc      *win32.Process
}

type Result struct {
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
	WorkingSetSizeBytes        uint64
	PeakWorkingSetSizeBytes    uint64
	PrivateUsageBytes          uint64
	PeakPagefileUsageBytes     uint64
	PeakPagedPoolUsageBytes    uint64
	PagedPoolUsageBytes        uint64
	PeakNonPagedPoolUsageBytes uint64
	NonPagedPoolUsageBytes     uint64
	PageFaultCount             uint64
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

func RunContained(cmd *exec.Cmd, cfg *Config) (*Container, error) {
	var container Container
	job, err := win32.CreateJobObject(cfg.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get create win32.JobObject")
	}
	container.Name = cfg.Name
	container.job = job
	token, err := win32.CurrentProcessToken()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get current process token")
	}
	logger := logWrapper{
		Logger: cfg.Logger,
	}
	if cfg.RestrictedToken {
		cfg.Logger.Logln("creating restricted token")
		rt, err := token.CreateRestrictedToken(win32.TokenRestrictions{
			DisableMaxPrivilege: true,
			LUAToken:            true,
			DisableSIDs: []string{
				"BUILTIN\\Administrator",
			},
		})
		logger.CloseLogError(token, "couldn't closed process token")
		if err != nil {
			return nil, errors.Wrapf(err, "unable to create restricted token")
		}
		token = rt
	}
	defer logger.CloseLogError(token, "couldn't closed process token")

	proc, err := win32.StartProcess(cmd, win32.AccessToken(token), win32.Suspended)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to start process")
	}
	if err = job.Assign(proc); err != nil {
		logger.Error(proc.Kill(), "unable to kill child process")
		return nil, err
	}
	container.proc = proc
	eli := &win32.ExtendedLimitInformation{
		KillOnJobClose: true,
	}
	if cfg.EnforceMemory {
		eli.JobMemoryLimit = MBToBytes * uint64(cfg.MemoryMBLimit)
	}

	if err = container.killOnError(job.SetInformation(eli)); err != nil {
		logger.CloseLogError(job, "failed to close JobObject")
		return nil, errors.Wrapf(err, "container: Could not set basic limit information")
	}
	if cfg.EnforceCPU {
		if cfg.CPUMHzLimit < MinimumCPUMHz {
			return nil, errors.Errorf("CPUMHzLimit is too low. Minimum is %d", MinimumCPUMHz)
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
				Rate:    win32.MHzToCPURate(uint64(cfg.CPUMHzLimit)),
			},
			Notify: true,
		}
		if err = container.killOnError(job.SetInformation(nli)); err != nil {
			logger.CloseLogError(job, "failed to close JobObject")
			return nil, errors.Wrapf(err, "container: Could not set cpu notification limits")
		}
		if err = container.killOnError(job.SetInformation(crci)); err != nil {
			logger.CloseLogError(job, "failed to close JobObject")
			return nil, errors.Wrapf(err, "container: Could not set cpu rate limits")
		}
	}
	if err = container.killOnError(proc.Resume()); err != nil {
		logger.CloseLogError(job, "failed to close JobObject")
		return nil, errors.Wrapf(err, "container: Could not resume process main thread")
	}
	container.exitCh = make(chan struct{})
	container.doneCh = make(chan struct{})
	return &container, nil
}

func (c *Container) PollViolations(fn func(v LimitViolation)) {
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
			for _, v := range violations {
				fn(v)
			}
		}
	}
}

func (c *Container) PollStats(fn func(stats ProcessStats)) {
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
			procTime := time.Since(c.StartTime)
			stats := ProcessStats{
				CPUStats: CPUStats{
					TotalRunTime:    procTime,
					TotalCPUTime:    procTime * time.Duration(runtime.NumCPU()),
					TotalKernelTime: info.Basic.TotalKernelTime,
					TotalUserTime:   info.Basic.TotalUserTime,
				},
				MemoryStats: MemoryStats{
					WorkingSetSizeBytes:        meminfo.WorkingSetSize,
					PeakWorkingSetSizeBytes:    meminfo.PeakWorkingSetSize,
					PrivateUsageBytes:          meminfo.PrivateUsage,
					PeakPagefileUsageBytes:     meminfo.PeakPagefileUsage,
					NonPagedPoolUsageBytes:     meminfo.QuotaNonPagedPoolUsage,
					PeakNonPagedPoolUsageBytes: meminfo.QuotaPeakNonPagedPoolUsage,
					PagedPoolUsageBytes:        meminfo.QuotaPagedPoolUsage,
					PeakPagedPoolUsageBytes:    meminfo.QuotaPeakPagedPoolUsage,
					PageFaultCount:             uint64(meminfo.PageFaultCount),
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
			fn(stats)
		}
	}
}

func (c *Container) Wait(exitCh <-chan struct{}) (Result, error) {
	pr, err := c.proc.Wait(exitCh)
	c.Logger.Logln(fmt.Sprintf("process exited: %d", pr.ExitStatus))
	if err != nil {
		return Result{}, err
	}
	return Result{
		ExitCode: pr.ExitStatus,
	}, pr.Err
}

func (c *Container) Shutdown(timeout time.Duration) error {
	return c.proc.Shutdown(timeout)
}

func (c *Container) killOnError(err error) error {
	if err != nil {
		c.Logger.Error(c.proc.Kill(), "unable to kill child process")
	}
	return err
}
