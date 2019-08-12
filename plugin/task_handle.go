package plugin

import (
	"context"
	"strconv"
	"sync"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/plugins/drivers"
	"github.com/jet/damon/container"
	"github.com/jet/damon/metrics"
	"github.com/jet/damon/win32"
)

type taskHandle struct {
	container *container.Container
	pid       int
	logger    hclog.Logger

	metrics *metrics.Metrics

	// stateLock syncs access to all fields below
	stateLock sync.RWMutex

	taskConfig  *drivers.TaskConfig
	procState   drivers.TaskState
	startedAt   time.Time
	completedAt time.Time
	exitResult  *drivers.ExitResult
}

func (h *taskHandle) TaskStatus() *drivers.TaskStatus {
	h.stateLock.RLock()
	defer h.stateLock.RUnlock()

	return &drivers.TaskStatus{
		ID:          h.taskConfig.ID,
		Name:        h.taskConfig.Name,
		State:       h.procState,
		StartedAt:   h.startedAt,
		CompletedAt: h.completedAt,
		ExitResult:  h.exitResult,
		DriverAttributes: map[string]string{
			"pid": strconv.Itoa(h.pid),
		},
	}
}

func (h *taskHandle) run() {
	resources := win32.GetSystemResources()
	m := &metrics.Metrics{
		Cores:      resources.CPUNumCores,
		MHzPerCore: resources.CPUMhzPercore,
	}
	m.Init()
	h.metrics = m
	h.logger.Debug("resources", "cores", resources.CPUNumCores, "mhz_per_core", resources.CPUMhzPercore)
	go h.container.PollStats(func(stats container.ProcessStats) {
		h.logger.Debug("container.PollStats", "stats", stats)
		m.OnStats(stats)
	})
	go h.container.PollViolations(m.OnViolation)
	res, err := h.container.WaitForResult(context.Background())
	completed := time.Now()
	if err != nil {
		h.stateLock.Lock()
		h.exitResult = &drivers.ExitResult{
			Err: err,
		}
		h.procState = drivers.TaskStateUnknown
		h.stateLock.Unlock()
		return
	}
	h.stateLock.Lock()
	h.procState = drivers.TaskStateExited
	h.exitResult = &drivers.ExitResult{
		ExitCode: res.ExitStatus,
	}
	h.completedAt = completed
	h.stateLock.Unlock()
}

func (h *taskHandle) IsRunning() bool {
	s := h.TaskStatus()
	return s.State != drivers.TaskStateExited
}

func (h *taskHandle) stats(ctx context.Context, interval time.Duration) (<-chan *drivers.TaskResourceUsage, error) {
	ch := make(chan *drivers.TaskResourceUsage)
	go h.handleStats(ctx, ch, interval)
	return ch, nil
}

var (
	measuredCPUStats    = []string{"System Mode", "User Mode", "Percent"}
	measuredMemoryStats = []string{"Usage", "Max Usage", "Kernel Usage", "Kernel Max Usage", "RSS"}
)

func (h *taskHandle) handleStats(ctx context.Context, ch chan *drivers.TaskResourceUsage, interval time.Duration) {
	defer close(ch)
	timer := time.NewTimer(0)
	for {
		select {
		case <-h.container.Done():
			return
		case <-ctx.Done():
			return
		case <-timer.C:
			timer.Reset(interval)
		}
		counters := h.metrics.PerfCounters()
		userTime := counters.CPUUserTime.TotalTime
		kerneltime := counters.CPUKernelTime.TotalTime
		totalTime := counters.CPUTotalTime
		var cpu drivers.CpuStats
		if totalTime > 0 {
			cpu = drivers.CpuStats{
				TotalTicks: float64(counters.CPUUserTime.Hz+counters.CPUKernelTime.Hz) / (1000000.0), // MHz
				SystemMode: float64(kerneltime / totalTime),
				UserMode:   float64(userTime / totalTime),
				Percent:    float64((userTime + kerneltime) / totalTime),
				Measured:   measuredCPUStats,
			}
		}
		// NOTE: I'm not entirely sure how to map the Windows memory counters to the linux-based memory stats (https://www.kernel.org/doc/Documentation/cgroup-v1/memory.txt)
		// The below is taken from the windows documentation: (see: https://docs.microsoft.com/en-us/windows/win32/memory/memory-performance-information)
		// and these are my best guesses about how they *might* map to their linux analogs:
		// - The "working set" is the amount of memory physically mapped to the process context at a given time. (usage)
		// - Memory in the paged pool is system memory that can be transferred to the paging file on disk (paged) when it is not being used. (memory.kmem.cache?)
		// - Memory in the nonpaged pool is system memory that cannot be paged to disk as long as the corresponding objects are allocated.  (memory.kmem.rss?)
		// - The pagefile usage represents how much memory is set aside for the process in the system paging file.
		//   The total amount of memory that the memory manager has committed for a running process.
		//
		// There are blog post about this, but I wasn't really able to extract the right details to make an acceptable guess:
		// - http://blogs.microsoft.co.il/sasha/2016/01/05/windows-process-memory-usage-demystified/
		// - https://blogs.msdn.microsoft.com/ericgolpe/2015/03/18/comparing-linuxunix-and-windows-performance-counters-on-microsoft-azure/
		//
		// From all this I gather that:
		// - Pool memory (NonPaged/Paged) is considered "Kernel Memory"
		// --  Seems likely we can add these two together to get the total kernel memory used by the process
		// - PrivateBytes is the total Virtual Memory used by the process that is not shareable.
		// --  Unfortunately I don't think you can do math with this and WSS to get RSS, because this does not include Shared memory
		// - Working Set is the In-Memory Private Bytes + Shared memory
		// --  probably the best approximation for "Usage" since it does not include paged memory.
		// --  MSFT annotates this as "Mem Usage" too.
		mem := drivers.MemoryStats{
			Usage:          counters.MemoryWorkingSetBytes,
			RSS:            counters.MemoryPrivateUsageBytes,
			MaxUsage:       counters.MemoryPeakWorkingSetBytes,
			KernelMaxUsage: counters.MemoryPeakPagedPoolUsageBytes + counters.MemoryPeakNonPagedPoolUsageBytes,
			KernelUsage:    counters.MemoryPagedPoolUsageBytes + counters.MemoryNonPagedPoolUsageBytes,
			Measured:       measuredMemoryStats,
		}
		usage := drivers.TaskResourceUsage{
			ResourceUsage: &drivers.ResourceUsage{
				CpuStats:    &cpu,
				MemoryStats: &mem,
			},
			Timestamp: time.Now().UTC().UnixNano(),
		}
		h.logger.Debug("task usage", "task_id", h.taskConfig.ID, "ts", usage.Timestamp, "cpu", cpu, "mem", mem)
		select {
		case <-h.container.Done():
			return
		case <-ctx.Done():
			return
		case ch <- &usage:
		}
	}
}

func (h *taskHandle) shutdown(timeout time.Duration) error {
	return h.container.Shutdown(timeout)
}
