package metrics

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jet/damon/container"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Namespace  string
	Labels     map[string]string
	MHzPerCore float64
	Cores      int

	cpuCollector *CPUCollector
	registry     *prometheus.Registry
	handler      http.Handler
	perfLock     sync.Mutex
	perfCounters atomic.Value

	// cpu
	cpuKernelTime    prometheus.Gauge
	cpuUserTime      prometheus.Gauge
	cpuKernelPercent prometheus.Gauge
	cpuUserPercent   prometheus.Gauge
	cpuKernelHz      prometheus.Gauge
	cpuUserHz        prometheus.Gauge
	cpuNotification  prometheus.Counter

	// memory
	memoryWorkingSet            prometheus.Gauge
	memoryCommitCharge          prometheus.Gauge
	memoryPageFaultCount        prometheus.Gauge
	memoryPeakWorkingSet        prometheus.Gauge
	memoryPeakPagefileUsage     prometheus.Gauge
	memoryPeakPagedPoolUsage    prometheus.Gauge
	memoryPagedPoolUsage        prometheus.Gauge
	memoryPeakNonPagedPoolUsage prometheus.Gauge
	memoryNonPagedPoolUsage     prometheus.Gauge
	memoryNotification          prometheus.Counter

	// io
	ioTxTotalBytes    prometheus.Gauge
	ioTxReadBytes     prometheus.Gauge
	ioTxWriteBytes    prometheus.Gauge
	ioTxOtherBytes    prometheus.Gauge
	ioReadOpsTotal    prometheus.Gauge
	ioWriteOpsTotal   prometheus.Gauge
	ioOtherOpsTotal   prometheus.Gauge
	ioTotalOperations prometheus.Gauge
	ioNotification    prometheus.Counter
}

func (m *Metrics) Init() {
	m.cpuCollector = &CPUCollector{
		MHzPerCore: m.MHzPerCore,
		Cores:      m.Cores,
	}
	m.registry = prometheus.NewRegistry()
	m.handler = promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
	m.cpuKernelTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "cpu",
		Name:        "kernel_seconds",
		Help:        `The number of seconds the process spent in kernel-mode`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.cpuKernelTime)
	m.cpuUserTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "cpu",
		Name:        "user_seconds",
		Help:        `The number of seconds the process spent in user-mode`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.cpuUserTime)
	m.cpuKernelPercent = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "cpu",
		Name:        "kernel_percent",
		Help:        `Percent of the total cpu time this process executed in kernel mode. This is calculated by measuring the total nanoseconds this process spend in kernel mode, and dividing it by the total available cpu time (cores * uptime)`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.cpuKernelPercent)
	m.cpuUserPercent = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "cpu",
		Name:        "user_percent",
		Help:        `Percent of the total cpu time this process executed in user mode.  This is calculated by measuring the total nanoseconds this process spend in user mode, and dividing it by the total available cpu time (cores * uptime)`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.cpuUserPercent)
	m.cpuKernelHz = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "cpu",
		Name:        "kernel_hz",
		Help:        `Kernel-mode time converted to Hz. This is calculated by taking the kernel percent and multiplying with the total available CPU hz (cores * hz per core)`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.cpuKernelHz)
	m.cpuUserHz = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "cpu",
		Name:        "user_hz",
		Help:        `User-mode time converted to Hz. This is calculated by taking the user percent and multiplying with the total available CPU hz (cores * hz per core)`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.cpuUserHz)
	m.cpuNotification = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   m.Namespace,
		Subsystem:   "cpu",
		Name:        "notifications_total",
		Help:        `Total number of CPU limit exceeded notifications.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.cpuNotification)
	m.memoryWorkingSet = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "memory",
		Name:        "working_set_bytes",
		Help:        `The current working set size, in bytes`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.memoryWorkingSet)
	m.memoryCommitCharge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "memory",
		Name:        "commit_charge_bytes",
		Help:        `The Commit Charge value in bytes for this process. Commit Charge is the total amount of memory that the memory manager has committed for a running process.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.memoryCommitCharge)
	m.memoryPeakPagefileUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "memory",
		Name:        "peak_pagefile_usage_bytes",
		Help:        `The peak value in bytes of the Commit Charge during the lifetime of this process.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.memoryPeakPagefileUsage)
	m.memoryPeakWorkingSet = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "memory",
		Name:        "peak_working_set_bytes",
		Help:        `The peak working set size, in bytes`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.memoryPeakWorkingSet)
	m.memoryPeakPagedPoolUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "memory",
		Name:        "quota_peak_paged_pool_usage",
		Help:        `The peak paged pool usage, in bytes.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.memoryPeakPagedPoolUsage)
	m.memoryPagedPoolUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "memory",
		Name:        "quota_nonpaged_pool_usage",
		Help:        `The current nonpaged pool usage, in bytes.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.memoryPagedPoolUsage)
	m.memoryPeakNonPagedPoolUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "memory",
		Name:        "quota_peak_nonpaged_pool_usage",
		Help:        `The peak nonpaged pool usage, in bytes.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.memoryPeakNonPagedPoolUsage)
	m.memoryNonPagedPoolUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "memory",
		Name:        "quota_paged_pool_usage",
		Help:        `The current paged pool usage, in bytes.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.memoryNonPagedPoolUsage)
	m.memoryPageFaultCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "memory",
		Name:        "page_fault_total",
		Help:        `The number of page faults.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.memoryPageFaultCount)
	m.memoryNotification = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   m.Namespace,
		Subsystem:   "memory",
		Name:        "notifications_total",
		Help:        `Total number of Memory limit exceeded notifications.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.memoryNotification)

	// io operations
	m.ioReadOpsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "io",
		Name:        "read_operations_total",
		Help:        `Total number of read IO operations.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.ioReadOpsTotal)
	m.ioWriteOpsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "io",
		Name:        "write_operations_total",
		Help:        `Total number of write IO operations.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.ioWriteOpsTotal)
	m.ioOtherOpsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "io",
		Name:        "other_operations_total",
		Help:        `Total number of other IO operations.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.ioOtherOpsTotal)
	m.ioTotalOperations = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "io",
		Name:        "operations_total",
		Help:        `Total number of IO operations.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.ioTotalOperations)
	// io bytes
	m.ioTxReadBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "io",
		Name:        "read_bytes",
		Help:        `Total number of IO read bytes transferred.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.ioTxReadBytes)
	m.ioTxWriteBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "io",
		Name:        "write_bytes",
		Help:        `Total number of IO write bytes transferred.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.ioTxWriteBytes)
	m.ioTxOtherBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "io",
		Name:        "other_bytes",
		Help:        `Total number of IO other bytes transferred.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.ioTxOtherBytes)
	m.ioTxTotalBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "io",
		Name:        "total_bytes",
		Help:        `Total number of IO bytes trasferred.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.ioTxTotalBytes)
	// io notifications
	m.ioNotification = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   m.Namespace,
		Subsystem:   "io",
		Name:        "notifications_total",
		Help:        `Total number of IO limit exceeded notifications.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.ioNotification)
	m.perfCounters.Store(PerfCounters{})
}

func (m *Metrics) OnStats(stats container.ProcessStats) {
	m.perfLock.Lock()
	defer m.perfLock.Unlock()
	counters := m.PerfCounters()
	counters.TimeStamp = time.Now()
	sample := m.cpuCollector.Sample(CPUMeasurement{
		TotalTime:  stats.CPUStats.TotalCPUTime,
		UserTime:   stats.CPUStats.TotalUserTime,
		KernelTime: stats.CPUStats.TotalKernelTime,
	})
	// cpu
	m.cpuUserTime.Set(stats.CPUStats.TotalUserTime.Seconds())
	m.cpuKernelTime.Set(stats.CPUStats.TotalKernelTime.Seconds())
	m.cpuKernelHz.Set(float64(sample.KernelHz))
	m.cpuKernelPercent.Set(sample.KernelPercent)
	m.cpuUserHz.Set(float64(sample.UserHz))
	m.cpuUserPercent.Set(sample.UserPercent)
	counters.CPUUserTime = CPUTime{
		TotalTime: stats.CPUStats.TotalUserTime,
		Hz:        sample.UserHz,
		Percent:   sample.UserPercent,
	}
	counters.CPUKernelTime = CPUTime{
		TotalTime: stats.CPUStats.TotalKernelTime,
		Hz:        sample.KernelHz,
		Percent:   sample.KernelPercent,
	}
	counters.CPUTotalTime = stats.CPUStats.TotalCPUTime
	// memory
	m.memoryCommitCharge.Set(float64(stats.MemoryStats.PrivateUsageBytes))
	m.memoryWorkingSet.Set(float64(stats.MemoryStats.WorkingSetSizeBytes))
	m.memoryPageFaultCount.Set(float64(stats.MemoryStats.PageFaultCount))
	m.memoryPeakPagefileUsage.Set(float64(stats.MemoryStats.PeakPagefileUsageBytes))
	m.memoryPeakWorkingSet.Set(float64(stats.MemoryStats.PeakWorkingSetSizeBytes))
	m.memoryPeakNonPagedPoolUsage.Set(float64(stats.MemoryStats.PeakNonPagedPoolUsageBytes))
	m.memoryNonPagedPoolUsage.Set(float64(stats.MemoryStats.NonPagedPoolUsageBytes))
	m.memoryPeakPagedPoolUsage.Set(float64(stats.MemoryStats.PeakPagedPoolUsageBytes))
	m.memoryPagedPoolUsage.Set(float64(stats.MemoryStats.PagedPoolUsageBytes))
	counters.MemoryPrivateUsageBytes = stats.MemoryStats.PrivateUsageBytes
	counters.MemoryPeakWorkingSetBytes = stats.MemoryStats.PeakWorkingSetSizeBytes
	counters.MemoryWorkingSetBytes = stats.MemoryStats.WorkingSetSizeBytes
	counters.MemoryPeakPagefileUsageBytes = stats.MemoryStats.PeakWorkingSetSizeBytes
	counters.MemoryPageFaults = stats.MemoryStats.PageFaultCount
	counters.MemoryPeakNonPagedPoolUsageBytes = stats.MemoryStats.PeakNonPagedPoolUsageBytes
	counters.MemoryNonPagedPoolUsageBytes = stats.MemoryStats.NonPagedPoolUsageBytes
	counters.MemoryPeakPagedPoolUsageBytes = stats.MemoryStats.PeakPagedPoolUsageBytes
	counters.MemoryPagedPoolUsageBytes = stats.MemoryStats.PagedPoolUsageBytes
	// io
	m.ioTxReadBytes.Set(float64(stats.IOStats.TotalTxReadBytes))
	m.ioTxWriteBytes.Set(float64(stats.IOStats.TotalTxWrittenBytes))
	m.ioTxOtherBytes.Set(float64(stats.IOStats.TotalTxOtherBytes))
	m.ioTxTotalBytes.Set(float64(stats.IOStats.TotalTxCountBytes))
	m.ioReadOpsTotal.Set(float64(stats.IOStats.TotalReadIOOperations))
	m.ioWriteOpsTotal.Set(float64(stats.IOStats.TotalWriteIOOperations))
	m.ioOtherOpsTotal.Set(float64(stats.IOStats.TotalOtherIOOperations))
	m.ioTotalOperations.Set(float64(stats.IOStats.TotalIOOperations))
	counters.IOTxReadBytes = stats.IOStats.TotalTxReadBytes
	counters.IOTxWriteBytes = stats.IOStats.TotalTxWrittenBytes
	counters.IOTxOtherBytes = stats.IOStats.TotalTxOtherBytes
	counters.IOTxTotalBytes = stats.IOStats.TotalTxCountBytes
	counters.IOReadOpsTotal = stats.IOStats.TotalReadIOOperations
	counters.IOWriteOpsTotal = stats.IOStats.TotalWriteIOOperations
	counters.IOOtherOpsTotal = stats.IOStats.TotalOtherIOOperations
	counters.IOTotalOperations = stats.IOStats.TotalIOOperations
	m.perfCounters.Store(counters)
}

func (m *Metrics) OnViolation(v container.LimitViolation) {
	m.perfLock.Lock()
	defer m.perfLock.Unlock()
	counters := m.PerfCounters()
	switch v.Type {
	case container.IOLimitViolation:
		m.ioNotification.Inc()
		counters.IOViolations++
	case container.CPULimitViolation:
		m.cpuNotification.Inc()
		counters.CPUViolations++
	case container.MemoryLimitViolation:
		m.memoryNotification.Inc()
		counters.MemoryViolations++
	}
	m.perfCounters.Store(counters)
}

func (m *Metrics) PerfCounters() PerfCounters {
	return (m.perfCounters.Load()).(PerfCounters)
}

type PerfCounters struct {
	TimeStamp time.Time
	// cpu
	CPUUserTime   CPUTime
	CPUKernelTime CPUTime
	CPUTotalTime  time.Duration
	CPUViolations uint64
	// memory
	MemoryPrivateUsageBytes          uint64
	MemoryPeakPagefileUsageBytes     uint64
	MemoryWorkingSetBytes            uint64
	MemoryPeakWorkingSetBytes        uint64
	MemoryPeakPagedPoolUsageBytes    uint64
	MemoryPagedPoolUsageBytes        uint64
	MemoryPeakNonPagedPoolUsageBytes uint64
	MemoryNonPagedPoolUsageBytes     uint64
	MemoryPageFaults                 uint64
	MemoryViolations                 uint64
	// io
	IOTxReadBytes     uint64
	IOTxWriteBytes    uint64
	IOTxOtherBytes    uint64
	IOTxTotalBytes    uint64
	IOReadOpsTotal    uint64
	IOWriteOpsTotal   uint64
	IOOtherOpsTotal   uint64
	IOTotalOperations uint64
	IOViolations      uint64
}

type CPUTime struct {
	TotalTime time.Duration
	Hz        uint64
	Percent   float64
}

func (m *Metrics) Handler() http.Handler {
	return m.handler
}

type CPUCollector struct {
	LastTotalDuration  time.Duration
	LastUserDuration   time.Duration
	LastKernelDuration time.Duration
	Cores              int
	MHzPerCore         float64
	lock               sync.Mutex
}

type CPUMeasurement struct {
	TotalTime  time.Duration
	UserTime   time.Duration
	KernelTime time.Duration
}

type CPUSample struct {
	Measurement     CPUMeasurement
	DeltaKernelTime time.Duration
	DeltaUserTime   time.Duration
	DeltaTotalTime  time.Duration
	KernelPercent   float64
	KernelHz        uint64
	UserPercent     float64
	UserHz          uint64
}

func (c *CPUCollector) Sample(m CPUMeasurement) CPUSample {
	c.lock.Lock()
	t0 := c.LastTotalDuration
	k0 := c.LastKernelDuration
	u0 := c.LastUserDuration
	c.LastTotalDuration = m.TotalTime
	c.LastKernelDuration = m.KernelTime
	c.LastUserDuration = m.UserTime
	c.lock.Unlock()

	// total cpu time = total time * num cores
	ttime := (m.TotalTime - t0)
	tmhz := c.MHzPerCore * float64(c.Cores)

	kperc := float64(m.KernelTime-k0) / float64(ttime)
	uperc := float64(m.UserTime-u0) / float64(ttime)

	mHzToHz := 1000000.0
	khz := uint64(kperc * mHzToHz * tmhz)
	uhz := uint64(uperc * mHzToHz * tmhz)

	return CPUSample{
		DeltaTotalTime:  m.TotalTime - t0,
		DeltaKernelTime: m.KernelTime - k0,
		DeltaUserTime:   m.UserTime - k0,
		KernelHz:        khz,
		KernelPercent:   kperc,
		UserHz:          uhz,
		UserPercent:     uperc,
		Measurement:     m,
	}
}
