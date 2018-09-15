// +build windows

package win32

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

const (
	MaxCPURate uint = 10000
	MinCPURate uint = 1
	MaxWeight  uint = 9
	MinWeight  uint = 1
)

// BasicLimitInformation ...
type BasicLimitInformation struct {
	SchedulingClass   uint
	PriorityClass     PriorityClass
	MinWorkingSetSize int64
	MaxWorkingSetSize int64
	ProcessorAffinity uint64
}

type PriorityClass uint32

const (
	IdlePriortyClass        = PriorityClass(_IDLE_PRIORITY_CLASS)
	BelowNormalPriortyClass = PriorityClass(_BELOW_NORMAL_PRIORITY_CLASS)
	NormalPriortyClass      = PriorityClass(_NORMAL_PRIORITY_CLASS)
	AboveNormalPriortyClass = PriorityClass(_ABOVE_NORMAL_PRIORITY_CLASS)
	HighPriortyClass        = PriorityClass(_HIGH_PRIORITY_CLASS)
)

func (i *BasicLimitInformation) info() _JOBOBJECT_BASIC_LIMIT_INFORMATION {
	var info _JOBOBJECT_BASIC_LIMIT_INFORMATION
	if i == nil {
		return info
	}
	if i.MaxWorkingSetSize > i.MinWorkingSetSize && i.MaxWorkingSetSize > 0 {
		info.LimitFlags |= _JOB_OBJECT_LIMIT_WORKINGSET
		info.MaximumWorkingSetSize = uintptr(i.MaxWorkingSetSize)
		info.MinimumWorkingSetSize = uintptr(i.MinWorkingSetSize)
	}
	if i.PriorityClass != 0 {
		info.LimitFlags |= _JOB_OBJECT_LIMIT_PRIORITY_CLASS
		info.PriorityClass = uint32(i.PriorityClass)
	}
	if i.SchedulingClass >= 1 && i.SchedulingClass <= 9 {
		info.SchedulingClass = uint32(i.SchedulingClass)
		info.LimitFlags |= _JOB_OBJECT_LIMIT_SCHEDULING_CLASS
	}
	if i.ProcessorAffinity != 0 {
		info.Affinity = uintptr(i.ProcessorAffinity)
		info.LimitFlags |= _JOB_OBJECT_LIMIT_AFFINITY
	}
	return info
}

func (i *BasicLimitInformation) SetJobInfo(hJob syscall.Handle) error {
	info := i.info()
	ret, _, err := procSetInformationJobObject.Call(
		uintptr(hJob),
		uintptr(_JobObjectBasicLimitInformation),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Sizeof(info)),
	)
	if ret == 0 {
		return err
	}
	return nil
}

func (i *BasicLimitInformation) Extended() *ExtendedLimitInformation {
	return &ExtendedLimitInformation{
		Basic: i,
	}
}

// ExtendedLimitInformation ...
type ExtendedLimitInformation struct {
	Basic              *BasicLimitInformation
	KillOnJobClose     bool
	BreakawayOK        bool
	SilentBreakawayOK  bool
	JobMemoryLimit     uint64
	ProcessMemoryLimit uint64
}

func (i *ExtendedLimitInformation) SetJobInfo(hJob syscall.Handle) error {
	var info _JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation = i.Basic.info()
	if i.JobMemoryLimit > 0 {
		info.JobMemoryLimit = uintptr(i.JobMemoryLimit)
		info.BasicLimitInformation.LimitFlags |= _JOB_OBJECT_LIMIT_JOB_MEMORY
	}
	if i.JobMemoryLimit > 0 {
		info.ProcessMemoryLimit = uintptr(i.ProcessMemoryLimit)
		info.BasicLimitInformation.LimitFlags |= _JOB_OBJECT_LIMIT_JOB_MEMORY
	}
	if i.KillOnJobClose {
		info.BasicLimitInformation.LimitFlags |= _JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	}
	if i.BreakawayOK {
		info.BasicLimitInformation.LimitFlags |= _JOB_OBJECT_LIMIT_BREAKAWAY_OK
	}
	if i.SilentBreakawayOK {
		info.BasicLimitInformation.LimitFlags |= _JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK
	}
	ret, _, err := procSetInformationJobObject.Call(
		uintptr(hJob),
		uintptr(_JobObjectExtendedLimitInformation),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Sizeof(info)),
	)
	if ret == 0 {
		return err
	}
	return nil
}

type CPURateControlInformation struct {
	Rate   *CPUMaxRateInformation
	Weight uint
	MinMax *CPURateMinMaxInformation
	Notify bool
}
type CPURateMinMaxInformation struct {
	MaxRate int
	MinRate int
}

type CPUMaxRateInformation struct {
	Rate    uint
	HardCap bool
}

func MHzToWeight(mhz uint64) uint {
	if mhz == 0 {
		return 0
	}
	sr := GetSystemResources()
	r := float64(float64(mhz) / sr.CPUTotalTicks)
	weight := uint(r * float64(MaxWeight))
	if weight > MaxWeight {
		return MaxWeight
	}
	if weight < MinWeight {
		return MinWeight
	}
	return weight
}

func MHzToCPURate(mhz uint64) uint {
	if mhz == 0 {
		return 0
	}
	sr := GetSystemResources()
	r := float64(float64(mhz) / sr.CPUTotalTicks)
	rate := uint(r * 10000.0)
	if rate > MaxCPURate {
		return MaxCPURate
	}
	if rate < MinCPURate {
		return MinCPURate
	}
	return rate
}

func (i *CPURateControlInformation) SetJobInfo(hJob syscall.Handle) error {
	var pInfo unsafe.Pointer
	var size uintptr
	if i.Rate != nil {
		var info _JOBOBJECT_CPU_RATE_CONTROL_INFORMATION
		size = unsafe.Sizeof(info)
		info.Rate = uint32(i.Rate.Rate)
		info.ControlFlags = JOB_OBJECT_CPU_RATE_CONTROL_ENABLE
		if i.Rate.HardCap {
			info.ControlFlags |= JOB_OBJECT_CPU_RATE_CONTROL_HARD_CAP
		}
		pInfo = unsafe.Pointer(&info)
	} else if i.Weight > 0 && i.Weight <= MaxWeight {
		var info _JOBOBJECT_CPU_RATE_CONTROL_INFORMATION
		size = unsafe.Sizeof(info)
		info.Rate = uint32(i.Weight)
		info.ControlFlags |= JOB_OBJECT_CPU_RATE_CONTROL_ENABLE | JOB_OBJECT_CPU_RATE_CONTROL_WEIGHT_BASED
		pInfo = unsafe.Pointer(&info)
	} else if i.MinMax != nil {
		var info _JOBOBJECT_CPU_RATE_CONTROL_INFORMATION_MINMAX
		info.MinRate = uint16(i.MinMax.MinRate)
		info.MaxRate = uint16(i.MinMax.MaxRate)
		info.ControlFlags = JOB_OBJECT_CPU_RATE_CONTROL_ENABLE | JOB_OBJECT_CPU_RATE_CONTROL_MIN_MAX_RATE
		pInfo = unsafe.Pointer(&info)
	}
	ret, _, err := procSetInformationJobObject.Call(
		uintptr(hJob),
		uintptr(_JobObjectCpuRateControlInformation),
		uintptr(pInfo),
		uintptr(size),
	)
	if ret == 0 {
		return err
	}
	return nil
}

/*type _JOBOBJECT_IO_RATE_CONTROL_INFORMATION struct {
	MaxIops         int64
	MaxBandwidth    int64
	ReservationIops int64
	VolumeName      *uint16
	BaseIoSize      uint32
	ControlFlags    uint32
}*/

type IORateControlInformation struct {
	MaxBandwidth int64
	ReservedIOPS int64
	MaxIOPS      int64
	BaseIOSize   uint32
	VolumeName   string
}

func (i *IORateControlInformation) SetJobInfo(hJob syscall.Handle) error {
	// Enable
	if i.MaxBandwidth > 0 || i.ReservedIOPS > 0 || i.MaxIOPS > 0 {
		info := _JOBOBJECT_IO_RATE_CONTROL_INFORMATION{
			MaxBandwidth:    i.MaxBandwidth,
			ReservationIops: i.ReservedIOPS,
			MaxIops:         i.MaxIOPS,
			VolumeName:      Text(i.VolumeName).WChars(),
			ControlFlags:    _JOB_OBJECT_IO_RATE_CONTROL_ENABLE,
		}
		return setIoRateControlInformationJobObject(hJob, info)
	}
	// Disable
	return setIoRateControlInformationJobObject(hJob, _JOBOBJECT_IO_RATE_CONTROL_INFORMATION{
		VolumeName:   Text(i.VolumeName).WChars(),
		ControlFlags: 0,
	})
}

func GetIORateControlInformations(job *JobObject, volume string) ([]IORateControlInformation, error) {
	is, err := queryIoRateControlInformationJobObject(job.hJob, volume)
	if err != nil {
		return nil, err
	}
	var infos []IORateControlInformation
	for _, i := range is {
		infos = append(infos, IORateControlInformation{
			MaxBandwidth: i.MaxBandwidth,
			MaxIOPS:      i.MaxIops,
			ReservedIOPS: i.ReservationIops,
			VolumeName:   UTF16PtrToString(i.VolumeName),
			BaseIOSize:   i.BaseIoSize,
		})
	}
	return infos, nil
}

type NetRateControlInformation struct {
	MaxBandwidth uint64
	DSCPTag      byte
}

func (i *NetRateControlInformation) SetJobInfo(hJob syscall.Handle) error {
	var info _JOBOBJECT_NET_RATE_CONTROL_INFORMATION
	if i.MaxBandwidth > 0 {
		info.MaxBandwidth = i.MaxBandwidth
		info.ControlFlags |= JOB_OBJECT_NET_RATE_CONTROL_ENABLE | JOB_OBJECT_NET_RATE_CONTROL_MAX_BANDWIDTH
	}
	if i.DSCPTag > 0 {
		info.DscpTag = (i.DSCPTag & 0x3F)
		info.ControlFlags |= JOB_OBJECT_NET_RATE_CONTROL_ENABLE | JOB_OBJECT_NET_RATE_CONTROL_DSCP_TAG
	}
	ret, _, err := procSetInformationJobObject.Call(
		uintptr(hJob),
		uintptr(_JobObjectNetRateControlInformation),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Sizeof(info)),
	)
	if ret == 0 {
		return err
	}
	return nil
}

type NotificationLimitInformation struct {
	UserTimeLimit     time.Duration
	CPURateLimit      *NotificationRateLimitTolerance
	IORateLimit       *NotificationRateLimitTolerance
	NetworkRateLimit  *NotificationRateLimitTolerance
	JobMemoryLimit    *NotificationMemoryLimit
	JobLowMemoryLimit *NotificationMemoryLimit
	IOReadBytesLimit  uint64
	IOWriteBytesLimit uint64
}

type NotificationRateLimitTolerance struct {
	Level    JobObjectRateControlTolerance
	Interval JobObjectRateControlToleranceInterval
}

type NotificationMemoryLimit struct {
	Bytes uint64
}

func (i *NotificationLimitInformation) SetJobInfo(hJob syscall.Handle) error {
	var info _JOBOBJECT_NOTIFICATION_LIMIT_INFORMATION_2
	if i.UserTimeLimit > 0 {
		info.LimitFlags |= _JOB_OBJECT_LIMIT_JOB_TIME
		info.PerJobUserTimeLimit = uint64(i.UserTimeLimit / 100) // 100ns ticks
	}
	if i.CPURateLimit != nil {
		info.LimitFlags |= _JOB_OBJECT_LIMIT_CPU_RATE_CONTROL
		info.RateControlTolerance = uint32(i.CPURateLimit.Level)
		info.RateControlToleranceInterval = uint32(i.CPURateLimit.Interval)
	}
	if i.IORateLimit != nil {
		info.LimitFlags |= _JOB_OBJECT_LIMIT_IO_RATE_CONTROL
		info.IoRateControlTolerance = uint32(i.IORateLimit.Level)
		info.IoRateControlToleranceInterval = uint32(i.IORateLimit.Interval)
	}
	if i.NetworkRateLimit != nil {
		info.LimitFlags |= _JOB_OBJECT_LIMIT_NET_RATE_CONTROL
		info.NetRateControlTolerance = uint32(i.NetworkRateLimit.Level)
		info.NetRateControlToleranceInterval = uint32(i.NetworkRateLimit.Interval)
	}
	if i.JobMemoryLimit != nil {
		info.JobMemoryLimit = i.JobMemoryLimit.Bytes
		info.LimitFlags |= _JOB_OBJECT_LIMIT_JOB_MEMORY
	}
	if i.JobLowMemoryLimit != nil {
		info.JobLowMemoryLimit = i.JobLowMemoryLimit.Bytes
		info.LimitFlags |= _JOB_OBJECT_LIMIT_JOB_MEMORY_LOW
	}
	if i.IOReadBytesLimit > 0 {
		info.IoReadBytesLimit = i.IOReadBytesLimit
		info.LimitFlags |= _JOB_OBJECT_LIMIT_JOB_READ_BYTES
	}
	if i.IOReadBytesLimit > 0 {
		info.IoWriteBytesLimit = i.IOWriteBytesLimit
		info.LimitFlags |= _JOB_OBJECT_LIMIT_JOB_WRITE_BYTES
	}

	ret, _, err := procSetInformationJobObject.Call(
		uintptr(hJob),
		uintptr(_JobObjectNotificationLimitInformation2),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Sizeof(info)),
	)
	if ret == 0 {
		fmt.Println("procSetInformationJobObject err", err)
		return err
	}
	return nil
}

// IOCounters operations and transfer counts
type IOCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

// JobObjectBasicAccounting basic accounting information for process time and number of processes
type JobObjectBasicAccounting struct {
	TotalUserTime             time.Duration
	TotalKernelTime           time.Duration
	ThisPeriodTotalUserTime   time.Duration
	ThisPeriodTotalKernelTime time.Duration
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32
}

type JobObjectBasicAndIOAccounting struct {
	Basic JobObjectBasicAccounting
	IO    IOCounters
}

func (info *JobObjectBasicAndIOAccounting) GetJobInfo(hJob syscall.Handle) error {
	i, err := queryBasicAndIOAccounting(hJob)
	if err != nil {
		return err
	}
	info.Basic = JobObjectBasicAccounting{
		TotalUserTime:             time.Duration(i.BasicInfo.TotalUserTime) * 100,
		TotalKernelTime:           time.Duration(i.BasicInfo.TotalKernelTime) * 100,
		ThisPeriodTotalUserTime:   time.Duration(i.BasicInfo.ThisPeriodTotalUserTime) * 100,
		ThisPeriodTotalKernelTime: time.Duration(i.BasicInfo.ThisPeriodTotalKernelTime) * 100,
		TotalPageFaultCount:       i.BasicInfo.TotalPageFaultCount,
		TotalProcesses:            i.BasicInfo.TotalProcesses,
		ActiveProcesses:           i.BasicInfo.ActiveProcesses,
		TotalTerminatedProcesses:  i.BasicInfo.TotalTerminatedProcesses,
	}
	info.IO = IOCounters{
		ReadOperationCount:  i.IoInfo.ReadOperationCount,
		WriteOperationCount: i.IoInfo.WriteOperationCount,
		OtherOperationCount: i.IoInfo.OtherOperationCount,
		ReadTransferCount:   i.IoInfo.ReadTransferCount,
		WriteTransferCount:  i.IoInfo.WriteTransferCount,
		OtherTransferCount:  i.IoInfo.OtherTransferCount,
	}
	return nil
}
