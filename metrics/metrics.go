package metrics

import (
	"net/http"
	"sync"
	"time"

	"github.com/jet/damon/container"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Namespace        string
	Labels           map[string]string
	MHzPerCore       float64
	Cores            int
	CPULimitHz       float64
	MemoryLimitBytes float64

	cpuCollector *CPUCollector
	registry     *prometheus.Registry
	handler      http.Handler

	// cpu
	cpuKernelTime    prometheus.Gauge
	cpuUserTime      prometheus.Gauge
	cpuKernelPercent prometheus.Gauge
	cpuUserPercent   prometheus.Gauge
	cpuKernelHz      prometheus.Gauge
	cpuUserHz        prometheus.Gauge
	cpuLimitHz       prometheus.Gauge
	cpuLimitPercent  prometheus.Gauge
	cpuNotification  prometheus.Counter

	// memory
	memoryWorkingSet     prometheus.Gauge
	memoryCommitCharge   prometheus.Gauge
	memoryPageFaultCount prometheus.Gauge
	memoryLimitBytes     prometheus.Gauge
	memoryNotification   prometheus.Counter

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
	m.cpuLimitHz = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "cpu",
		Name:        "limit_hz",
		Help:        "The configured CPU usage limit in Hz.",
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.cpuLimitHz)
	m.cpuLimitPercent = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "cpu",
		Name:        "limit_percent",
		Help:        "The configured CPU usage limit as a percentage of total system Hz available.",
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.cpuLimitPercent)
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
	m.memoryPageFaultCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "memory",
		Name:        "page_fault_total",
		Help:        `The number of page faults.`,
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.memoryPageFaultCount)
	m.memoryLimitBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.Namespace,
		Subsystem:   "memory",
		Name:        "limit_bytes",
		Help:        "The configured Memory limit in bytes.",
		ConstLabels: prometheus.Labels(m.Labels),
	})
	m.registry.MustRegister(m.memoryLimitBytes)
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
}

func (m *Metrics) OnStats(stats container.ProcessStats) {
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
	m.cpuLimitHz.Set(m.CPULimitHz)
	m.cpuLimitPercent.Set(m.CPULimitHz / (m.MHzPerCore * float64(m.Cores) * 1000000.0))
	// memory
	m.memoryCommitCharge.Set(float64(stats.MemoryStats.PrivateUsageBytes))
	m.memoryWorkingSet.Set(float64(stats.MemoryStats.WorkingSetSizeBytes))
	m.memoryPageFaultCount.Set(float64(stats.MemoryStats.PageFaultCount))
	m.memoryLimitBytes.Set(m.MemoryLimitBytes)
	// io
	m.ioTxReadBytes.Set(float64(stats.IOStats.TotalTxReadBytes))
	m.ioTxWriteBytes.Set(float64(stats.IOStats.TotalTxWrittenBytes))
	m.ioTxOtherBytes.Set(float64(stats.IOStats.TotalTxOtherBytes))
	m.ioTxTotalBytes.Set(float64(stats.IOStats.TotalTxCountBytes))
	m.ioReadOpsTotal.Set(float64(stats.IOStats.TotalReadIOOperations))
	m.ioWriteOpsTotal.Set(float64(stats.IOStats.TotalWriteIOOperations))
	m.ioOtherOpsTotal.Set(float64(stats.IOStats.TotalOtherIOOperations))
	m.ioTotalOperations.Set(float64(stats.IOStats.TotalIOOperations))
}

func (m *Metrics) OnViolation(v container.LimitViolation) {
	switch v.Type {
	case container.IOLimitViolation:
		m.ioNotification.Inc()
	case container.CPULimitViolation:
		m.cpuNotification.Inc()
	case container.MemoryLimitViolation:
		m.memoryNotification.Inc()
	}
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
	ttime := (m.TotalTime - t0) * time.Duration(c.Cores)
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
