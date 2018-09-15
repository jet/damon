// +build windows

package win32

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

var (
	procQueryInformationJobObject              = kernel32DLL.NewProc("QueryInformationJobObject")
	procSetInformationJobObject                = kernel32DLL.NewProc("SetInformationJobObject")
	procQueryIoRateControlInformationJobObject = kernel32DLL.NewProc("QueryIoRateControlInformationJobObject")
	procSetIoRateControlInformationJobObject   = kernel32DLL.NewProc("SetIoRateControlInformationJobObject")
	procFreeMemoryJobObject                    = kernel32DLL.NewProc("FreeMemoryJobObject")
)

const (
	_JOB_OBJECT_LIMIT_WORKINGSET                 uint32 = 0x00000001
	_JOB_OBJECT_LIMIT_PROCESS_TIME               uint32 = 0x00000002
	_JOB_OBJECT_LIMIT_JOB_TIME                   uint32 = 0x00000004
	_JOB_OBJECT_LIMIT_ACTIVE_PROCESS             uint32 = 0x00000008
	_JOB_OBJECT_LIMIT_AFFINITY                   uint32 = 0x00000010
	_JOB_OBJECT_LIMIT_PRIORITY_CLASS             uint32 = 0x00000020
	_JOB_OBJECT_LIMIT_PRESERVE_JOB_TIME          uint32 = 0x00000040
	_JOB_OBJECT_LIMIT_SCHEDULING_CLASS           uint32 = 0x00000080
	_JOB_OBJECT_LIMIT_PROCESS_MEMORY             uint32 = 0x00000100
	_JOB_OBJECT_LIMIT_JOB_MEMORY                 uint32 = 0x00000200
	_JOB_OBJECT_LIMIT_JOB_MEMORY_HIGH            uint32 = 0x00000200
	_JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION uint32 = 0x00000400
	_JOB_OBJECT_LIMIT_BREAKAWAY_OK               uint32 = 0x00000800
	_JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK        uint32 = 0x00001000
	_JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE          uint32 = 0x00002000
	_JOB_OBJECT_LIMIT_SUBSET_AFFINITY            uint32 = 0x00004000
	_JOB_OBJECT_LIMIT_JOB_MEMORY_LOW             uint32 = 0x00008000
	_JOB_OBJECT_LIMIT_READ_BYTES                 uint32 = 0x00010000
	_JOB_OBJECT_LIMIT_JOB_READ_BYTES             uint32 = 0x00010000
	_JOB_OBJECT_LIMIT_WRITE_BYTES                uint32 = 0x00020000
	_JOB_OBJECT_LIMIT_JOB_WRITE_BYTES            uint32 = 0x00020000
	_JOB_OBJECT_LIMIT_RATE_CONTROL               uint32 = 0x00040000
	_JOB_OBJECT_LIMIT_CPU_RATE_CONTROL           uint32 = 0x00040000
	_JOB_OBJECT_LIMIT_IO_RATE_CONTROL            uint32 = 0x00080000
	_JOB_OBJECT_LIMIT_NET_RATE_CONTROL           uint32 = 0x00100000
)

// reference
// https://www.codemachine.com/downloads/win10th2/winnt.h

const (
	JOB_OBJECT_CPU_RATE_CONTROL_ENABLE       = 0x1
	JOB_OBJECT_CPU_RATE_CONTROL_WEIGHT_BASED = 0x2
	JOB_OBJECT_CPU_RATE_CONTROL_HARD_CAP     = 0x4
	JOB_OBJECT_CPU_RATE_CONTROL_NOTIFY       = 0x8
	JOB_OBJECT_CPU_RATE_CONTROL_MIN_MAX_RATE = 0x10
)

const (
	JOB_OBJECT_NET_RATE_CONTROL_ENABLE        = 0x1
	JOB_OBJECT_NET_RATE_CONTROL_MAX_BANDWIDTH = 0x2
	JOB_OBJECT_NET_RATE_CONTROL_DSCP_TAG      = 0x4
	JOB_OBJECT_NET_RATE_CONTROL_VALID_FLAGS   = 0x7
)

const (
	_JOB_OBJECT_IO_RATE_CONTROL_ENABLE = 0x1
)

const (
	// do not reorder
	_JobObjectBasicAccountingInformation uint32 = iota + 1
	_JobObjectBasicLimitInformation
	_JobObjectBasicProcessIdList
	_JobObjectBasicUIRestrictions
	_JobObjectSecurityLimitInformation // deprecated
	_JobObjectEndOfJobTimeInformation
	_JobObjectAssociateCompletionPortInformation
	_JobObjectBasicAndIoAccountingInformation
	_JobObjectExtendedLimitInformation
	_JobObjectJobSetInformation
	_JobObjectGroupInformation
	_JobObjectNotificationLimitInformation
	_JobObjectLimitViolationInformation
	_JobObjectGroupInformationEx
	_JobObjectCpuRateControlInformation
	_JobObjectCompletionFilter
	_JobObjectCompletionCounter
	_JobObjectReserved1Information
	_JobObjectReserved2Information
	_JobObjectReserved3Information
	_JobObjectReserved4Information
	_JobObjectReserved5Information
	_JobObjectReserved6Information
	_JobObjectReserved7Information
	_JobObjectReserved8Information
	_JobObjectReserved9Information
	_JobObjectReserved10Information
	_JobObjectReserved11Information
	_JobObjectReserved12Information
	_JobObjectReserved13Information
	_JobObjectReserved14Information
	_JobObjectNetRateControlInformation
	_JobObjectNotificationLimitInformation2
	_JobObjectLimitViolationInformation2
	_JobObjectCreateSilo
	_JobObjectSiloBasicInformation
	_JobObjectReserved15Information
	_JobObjectReserved16Information
	_JobObjectReserved17Information
	_JobObjectReserved18Information
	_JobObjectReserved19Information
	_JobObjectReserved20Information
	_JobObjectReserved21Information
	_JobObjectReserved22Information
	_JobObjectReserved23Information
	_JobObjectReserved24Information
	_JobObjectReserved25Information
	_MaxJobObjectInfoClass
)

// typedef struct _JOBOBJECT_BASIC_LIMIT_INFORMATION {
//   LARGE_INTEGER PerProcessUserTimeLimit;
//   LARGE_INTEGER PerJobUserTimeLimit;
//   DWORD         LimitFlags;
//   SIZE_T        MinimumWorkingSetSize;
//   SIZE_T        MaximumWorkingSetSize;
//   DWORD         ActiveProcessLimit;
//   ULONG_PTR     Affinity;
//   DWORD         PriorityClass;
//   DWORD         SchedulingClass;
// }
type _JOBOBJECT_BASIC_LIMIT_INFORMATION struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

// typedef struct _JOBOBJECT_NOTIFICATION_LIMIT_INFORMATION {
//   DWORD64                                   IoReadBytesLimit;
//   DWORD64                                   IoWriteBytesLimit;
//   LARGE_INTEGER                             PerJobUserTimeLimit;
//   DWORD64                                   JobMemoryLimit;
//   JOBOBJECT_RATE_CONTROL_TOLERANCE          RateControlTolerance;
//   JOBOBJECT_RATE_CONTROL_TOLERANCE_INTERVAL RateControlToleranceInterval;
//   DWORD                                     LimitFlags;
// } JOBOBJECT_NOTIFICATION_LIMIT_INFORMATION, *PJOBOBJECT_NOTIFICATION_LIMIT_INFORMATION;
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-_jobobject_notification_limit_information
type _JOBOBJECT_NOTIFICATION_LIMIT_INFORMATION struct {
	IoReadBytesLimit             uint64
	IoWriteBytesLimit            uint64
	PerJobUserTimeLimit          uint64
	JobMemoryLimit               uint64
	RateControlTolerance         uint32
	RateControlToleranceInterval uint32
	LimitFlags                   uint32
	_                            [4]byte // pad
}

// typedef struct JOBOBJECT_NOTIFICATION_LIMIT_INFORMATION_2 {
//   DWORD64                                   IoReadBytesLimit;
//   DWORD64                                   IoWriteBytesLimit;
//   LARGE_INTEGER                             PerJobUserTimeLimit;
//   union {
//     DWORD64 JobHighMemoryLimit;
//     DWORD64 JobMemoryLimit;
//   } DUMMYUNIONNAME;
//   union {
//     JOBOBJECT_RATE_CONTROL_TOLERANCE RateControlTolerance;
//     JOBOBJECT_RATE_CONTROL_TOLERANCE CpuRateControlTolerance;
//   } DUMMYUNIONNAME2;
//   union {
//     JOBOBJECT_RATE_CONTROL_TOLERANCE_INTERVAL RateControlToleranceInterval;
//     JOBOBJECT_RATE_CONTROL_TOLERANCE_INTERVAL CpuRateControlToleranceInterval;
//   } DUMMYUNIONNAME3;
//   DWORD                                     LimitFlags;
//   JOBOBJECT_RATE_CONTROL_TOLERANCE          IoRateControlTolerance;
//   DWORD64                                   JobLowMemoryLimit;
//   JOBOBJECT_RATE_CONTROL_TOLERANCE_INTERVAL IoRateControlToleranceInterval;
//   JOBOBJECT_RATE_CONTROL_TOLERANCE          NetRateControlTolerance;
//   JOBOBJECT_RATE_CONTROL_TOLERANCE_INTERVAL NetRateControlToleranceInterval;
// };
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_notification_limit_information_2
type _JOBOBJECT_NOTIFICATION_LIMIT_INFORMATION_2 struct {
	IoReadBytesLimit                uint64
	IoWriteBytesLimit               uint64
	PerJobUserTimeLimit             uint64
	JobMemoryLimit                  uint64
	RateControlTolerance            uint32
	RateControlToleranceInterval    uint32
	LimitFlags                      uint32
	IoRateControlTolerance          uint32
	JobLowMemoryLimit               uint64
	IoRateControlToleranceInterval  uint32
	NetRateControlTolerance         uint32
	NetRateControlToleranceInterval uint32
	_                               [4]byte //pad
}

// typedef struct _JOBOBJECT_LIMIT_VIOLATION_INFORMATION {
//   DWORD                            LimitFlags;
//   DWORD                            ViolationLimitFlags;
//   DWORD64                          IoReadBytes;
//   DWORD64                          IoReadBytesLimit;
//   DWORD64                          IoWriteBytes;
//   DWORD64                          IoWriteBytesLimit;
//   LARGE_INTEGER                    PerJobUserTime;
//   LARGE_INTEGER                    PerJobUserTimeLimit;
//   DWORD64                          JobMemory;
//   DWORD64                          JobMemoryLimit;
//   JOBOBJECT_RATE_CONTROL_TOLERANCE RateControlTolerance;
//   JOBOBJECT_RATE_CONTROL_TOLERANCE RateControlToleranceLimit;
// } JOBOBJECT_LIMIT_VIOLATION_INFORMATION, *PJOBOBJECT_LIMIT_VIOLATION_INFORMATION;
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-_jobobject_limit_violation_information
type _JOBOBJECT_LIMIT_VIOLATION_INFORMATION struct {
	LimitFlags                uint32
	ViolationLimitFlags       uint32
	IoReadBytes               uint64
	IoReadBytesLimit          uint64
	IoWriteBytes              uint64
	IoWriteBytesLimit         uint64
	PerJobUserTime            uint64
	PerJobUserTimeLimit       uint64
	JobMemory                 uint64
	JobMemoryLimit            uint64
	RateControlTolerance      JobObjectRateControlTolerance
	RateControlToleranceLimit JobObjectRateControlTolerance
}

func (i *_JOBOBJECT_LIMIT_VIOLATION_INFORMATION) LimitViolationInfo() *LimitViolationInfo {
	info := &LimitViolationInfo{}
	f := uint32(i.LimitFlags)
	v := uint32(i.ViolationLimitFlags) & f
	if (v & _JOB_OBJECT_LIMIT_JOB_MEMORY_HIGH) > 0 {
		info.HighMemoryViolation = &LimitViolation{
			Measured: uint64(i.JobMemory),
			Limit:    uint64(i.JobMemoryLimit),
		}
	}
	if (v & _JOB_OBJECT_LIMIT_READ_BYTES) > 0 {
		info.IOReadBytesViolation = &LimitViolation{
			Measured: uint64(i.IoReadBytes),
			Limit:    uint64(i.IoReadBytesLimit),
		}
	}
	if (v & _JOB_OBJECT_LIMIT_WRITE_BYTES) > 0 {
		info.IOReadBytesViolation = &LimitViolation{
			Measured: uint64(i.IoWriteBytes),
			Limit:    uint64(i.IoWriteBytesLimit),
		}
	}
	if (v & _JOB_OBJECT_LIMIT_JOB_TIME) > 0 {
		info.JobTimeViolation = &LimitViolation{
			Measured: uint64(i.PerJobUserTime),
			Limit:    uint64(i.PerJobUserTimeLimit),
		}
	}
	if (v & _JOB_OBJECT_LIMIT_RATE_CONTROL) > 0 {
		info.CPURateViolation = &LimitViolation{
			Measured: uint64(i.RateControlTolerance),
			Limit:    uint64(i.RateControlToleranceLimit),
		}
	}
	return info
}

// typedef struct JOBOBJECT_LIMIT_VIOLATION_INFORMATION_2 {
//   DWORD                            LimitFlags;
//   DWORD                            ViolationLimitFlags;
//   DWORD64                          IoReadBytes;
//   DWORD64                          IoReadBytesLimit;
//   DWORD64                          IoWriteBytes;
//   DWORD64                          IoWriteBytesLimit;
//   LARGE_INTEGER                    PerJobUserTime;
//   LARGE_INTEGER                    PerJobUserTimeLimit;
//   DWORD64                          JobMemory;
//   union {
//     DWORD64 JobHighMemoryLimit;
//     DWORD64 JobMemoryLimit;
//   } DUMMYUNIONNAME;
//   union {
//     JOBOBJECT_RATE_CONTROL_TOLERANCE RateControlTolerance;
//     JOBOBJECT_RATE_CONTROL_TOLERANCE CpuRateControlTolerance;
//   } DUMMYUNIONNAME2;
//   union {
//     JOBOBJECT_RATE_CONTROL_TOLERANCE RateControlToleranceLimit;
//     JOBOBJECT_RATE_CONTROL_TOLERANCE CpuRateControlToleranceLimit;
//   } DUMMYUNIONNAME3;
//   DWORD64                          JobLowMemoryLimit;
//   JOBOBJECT_RATE_CONTROL_TOLERANCE IoRateControlTolerance;
//   JOBOBJECT_RATE_CONTROL_TOLERANCE IoRateControlToleranceLimit;
//   JOBOBJECT_RATE_CONTROL_TOLERANCE NetRateControlTolerance;
//   JOBOBJECT_RATE_CONTROL_TOLERANCE NetRateControlToleranceLimit;
// };
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_limit_violation_information_2
type _JOBOBJECT_LIMIT_VIOLATION_INFORMATION_2 struct {
	LimitFlags                   uint32
	ViolationLimitFlags          uint32
	IoReadBytes                  uint64
	IoReadBytesLimit             uint64
	IoWriteBytes                 uint64
	IoWriteBytesLimit            uint64
	PerJobUserTime               uint64
	PerJobUserTimeLimit          uint64
	JobMemory                    uint64
	JobMemoryLimit               uint64
	RateControlTolerance         uint32
	RateControlToleranceLimit    uint32
	JobLowMemoryLimit            uint64
	IORateControlTolerance       uint32
	IORateControlToleranceLimit  uint32
	NetRateControlTolerance      uint32
	NetRateControlToleranceLimit uint32
}

func (i *_JOBOBJECT_LIMIT_VIOLATION_INFORMATION_2) LimitViolationInfo() *LimitViolationInfo {
	info := &LimitViolationInfo{}
	f := uint32(i.LimitFlags)
	v := uint32(i.ViolationLimitFlags) & f
	if (v & _JOB_OBJECT_LIMIT_JOB_MEMORY_HIGH) > 0 {
		info.HighMemoryViolation = &LimitViolation{
			Measured: uint64(i.JobMemory),
			Limit:    uint64(i.JobMemoryLimit),
		}
	}
	if (v & _JOB_OBJECT_LIMIT_JOB_MEMORY_LOW) > 0 {
		info.HighMemoryViolation = &LimitViolation{
			Measured: uint64(i.JobMemory),
			Limit:    uint64(i.JobLowMemoryLimit),
		}
	}
	if (v & _JOB_OBJECT_LIMIT_READ_BYTES) > 0 {
		info.IOReadBytesViolation = &LimitViolation{
			Measured: uint64(i.IoReadBytes),
			Limit:    uint64(i.IoReadBytesLimit),
		}
	}
	if (v & _JOB_OBJECT_LIMIT_WRITE_BYTES) > 0 {
		info.IOReadBytesViolation = &LimitViolation{
			Measured: uint64(i.IoWriteBytes),
			Limit:    uint64(i.IoWriteBytesLimit),
		}
	}
	if (v & _JOB_OBJECT_LIMIT_JOB_TIME) > 0 {
		info.JobTimeViolation = &LimitViolation{
			Measured: uint64(i.PerJobUserTime),
			Limit:    uint64(i.PerJobUserTimeLimit),
		}
	}
	if (v & _JOB_OBJECT_LIMIT_RATE_CONTROL) > 0 {
		info.CPURateViolation = &LimitViolation{
			Measured: uint64(i.RateControlTolerance),
			Limit:    uint64(i.RateControlToleranceLimit),
		}
	}
	if (v & _JOB_OBJECT_LIMIT_IO_RATE_CONTROL) > 0 {
		info.IORateViolation = &LimitViolation{
			Measured: uint64(i.IORateControlTolerance),
			Limit:    uint64(i.IORateControlToleranceLimit),
		}
	}
	if (v & _JOB_OBJECT_LIMIT_NET_RATE_CONTROL) > 0 {
		info.IORateViolation = &LimitViolation{
			Measured: uint64(i.NetRateControlTolerance),
			Limit:    uint64(i.NetRateControlToleranceLimit),
		}
	}
	return info
}

// typedef struct _IO_COUNTERS {
//   ULONGLONG ReadOperationCount;
//   ULONGLONG WriteOperationCount;
//   ULONGLONG OtherOperationCount;
//   ULONGLONG ReadTransferCount;
//   ULONGLONG WriteTransferCount;
//   ULONGLONG OtherTransferCount;
// } IO_COUNTERS;
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-_io_counters
type _IO_COUNTERS struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

// typedef struct _JOBOBJECT_EXTENDED_LIMIT_INFORMATION {
//   JOBOBJECT_BASIC_LIMIT_INFORMATION BasicLimitInformation;
//   IO_COUNTERS                       IoInfo;
//   SIZE_T                            ProcessMemoryLimit;
//   SIZE_T                            JobMemoryLimit;
//   SIZE_T                            PeakProcessMemoryUsed;
//   SIZE_T                            PeakJobMemoryUsed;
// } JOBOBJECT_EXTENDED_LIMIT_INFORMATION, *PJOBOBJECT_EXTENDED_LIMIT_INFORMATION;
type _JOBOBJECT_EXTENDED_LIMIT_INFORMATION struct {
	BasicLimitInformation _JOBOBJECT_BASIC_LIMIT_INFORMATION
	IoInfo                _IO_COUNTERS
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

// typedef struct _JOBOBJECT_CPU_RATE_CONTROL_INFORMATION {
//   DWORD ControlFlags;
//   union {
//     DWORD CpuRate;
//     DWORD Weight;
//     struct {
//       WORD MinRate;
//       WORD MaxRate;
//     } DUMMYSTRUCTNAME;
//   } DUMMYUNIONNAME;
// } JOBOBJECT_CPU_RATE_CONTROL_INFORMATION, *PJOBOBJECT_CPU_RATE_CONTROL_INFORMATION;
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-_jobobject_cpu_rate_control_information
type _JOBOBJECT_CPU_RATE_CONTROL_INFORMATION struct {
	ControlFlags uint32
	Rate         uint32
}

type _JOBOBJECT_CPU_RATE_CONTROL_INFORMATION_MINMAX struct {
	ControlFlags uint32
	MinRate      uint16
	MaxRate      uint16
}

type JobObjectRateControlTolerance uint8

const (
	// do not reorder
	ToleranceLow JobObjectRateControlTolerance = iota + 1
	ToleranceMedium
	ToleranceHigh
)

type JobObjectRateControlToleranceInterval uint8

const (
	// do not reorder
	ToleranceIntervalShort JobObjectRateControlToleranceInterval = iota + 1
	ToleranceIntervalMedium
	ToleranceIntervalLong
)

func (t JobObjectRateControlTolerance) String() string {
	return fmt.Sprintf("%.2f%%", tolerenceLevelToPercent[t])
}

func (t JobObjectRateControlToleranceInterval) String() string {
	return fmt.Sprintf("%v", toleranceIntervalToDuration[t])
}

// map from TolerenceLevel to Percent (0-100.0)
var tolerenceLevelToPercent = map[JobObjectRateControlTolerance]float64{
	ToleranceLow:    20.0,
	ToleranceMedium: 40.0,
	ToleranceHigh:   60.0,
}

// map from ToleranceInterval to Duration
var toleranceIntervalToDuration = map[JobObjectRateControlToleranceInterval]time.Duration{
	ToleranceIntervalShort:  10 * time.Second,
	ToleranceIntervalMedium: time.Minute,
	ToleranceIntervalLong:   10 * time.Minute,
}

// typedef struct _JOBOBJECT_ASSOCIATE_COMPLETION_PORT {
// 	PVOID  CompletionKey;
// 	HANDLE CompletionPort;
//   } JOBOBJECT_ASSOCIATE_COMPLETION_PORT, *PJOBOBJECT_ASSOCIATE_COMPLETION_PORT;
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-_jobobject_associate_completion_port
type _JOBOBJECT_ASSOCIATE_COMPLETION_PORT struct {
	CompletionKey  uintptr
	CompletionPort syscall.Handle
}

// typedef struct JOBOBJECT_NET_RATE_CONTROL_INFORMATION {
//   DWORD64                           MaxBandwidth;
//   JOB_OBJECT_NET_RATE_CONTROL_FLAGS ControlFlags;
//   BYTE                              DscpTag;
// };
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-jobobject_net_rate_control_information
type _JOBOBJECT_NET_RATE_CONTROL_INFORMATION struct {
	MaxBandwidth uint64
	ControlFlags uint32
	DscpTag      byte
	_            [3]byte // pad
}

// typedef struct _JOBOBJECT_BASIC_ACCOUNTING_INFORMATION {
//     LARGE_INTEGER TotalUserTime;
//     LARGE_INTEGER TotalKernelTime;
//     LARGE_INTEGER ThisPeriodTotalUserTime;
//     LARGE_INTEGER ThisPeriodTotalKernelTime;
//     DWORD TotalPageFaultCount;
//     DWORD TotalProcesses;
//     DWORD ActiveProcesses;
//     DWORD TotalTerminatedProcesses;
// } JOBOBJECT_BASIC_ACCOUNTING_INFORMATION, *PJOBOBJECT_BASIC_ACCOUNTING_INFORMATION;
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-_jobobject_basic_accounting_information
type _JOBOBJECT_BASIC_ACCOUNTING_INFORMATION struct {
	TotalUserTime             uint64
	TotalKernelTime           uint64
	ThisPeriodTotalUserTime   uint64
	ThisPeriodTotalKernelTime uint64
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32
}

// typedef struct _JOBOBJECT_BASIC_AND_IO_ACCOUNTING_INFORMATION {
//   JOBOBJECT_BASIC_ACCOUNTING_INFORMATION BasicInfo;
//   IO_COUNTERS                            IoInfo;
// } JOBOBJECT_BASIC_AND_IO_ACCOUNTING_INFORMATION, *PJOBOBJECT_BASIC_AND_IO_ACCOUNTING_INFORMATION;
// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ns-winnt-_jobobject_basic_and_io_accounting_information
type _JOBOBJECT_BASIC_AND_IO_ACCOUNTING_INFORMATION struct {
	BasicInfo _JOBOBJECT_BASIC_ACCOUNTING_INFORMATION
	IoInfo    _IO_COUNTERS
}

// BOOL WINAPI QueryInformationJobObject(
// 	_In_opt_  HANDLE             hJob,
// 	_In_      JOBOBJECTINFOCLASS JobObjectInfoClass,
// 	_Out_     LPVOID             lpJobObjectInfo,
// 	_In_      DWORD              cbJobObjectInfoLength,
// 	_Out_opt_ LPDWORD            lpReturnLength
//   );
// https://msdn.microsoft.com/en-us/d843d578-fd67-4708-959f-00245ff70ec6

func queryBasicAndIOAccounting(hJob syscall.Handle) (*_JOBOBJECT_BASIC_AND_IO_ACCOUNTING_INFORMATION, error) {
	var info _JOBOBJECT_BASIC_AND_IO_ACCOUNTING_INFORMATION
	ret, _, err := procQueryInformationJobObject.Call(
		uintptr(hJob),
		uintptr(_JobObjectBasicAndIoAccountingInformation),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Sizeof(info)),
		uintptr(0),
	)
	if ret == 0 {
		return nil, err
	}
	return &info, nil
}

func queryJobObjectLimitViolationInformation(hJob syscall.Handle) (*_JOBOBJECT_LIMIT_VIOLATION_INFORMATION, error) {
	var info _JOBOBJECT_LIMIT_VIOLATION_INFORMATION
	ret, _, err := procQueryInformationJobObject.Call(
		uintptr(hJob),
		uintptr(_JobObjectLimitViolationInformation),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Sizeof(info)),
		uintptr(0),
	)
	if ret == 0 {
		return nil, err
	}
	return &info, nil
}

func queryJobObjectLimitViolationInformation2(hJob syscall.Handle) (*_JOBOBJECT_LIMIT_VIOLATION_INFORMATION_2, error) {
	var info _JOBOBJECT_LIMIT_VIOLATION_INFORMATION_2
	ret, _, err := procQueryInformationJobObject.Call(
		uintptr(hJob),
		uintptr(_JobObjectLimitViolationInformation2),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Sizeof(info)),
		uintptr(0),
	)
	if ret == 0 {
		return nil, err
	}
	return &info, nil
}

// typedef struct JOBOBJECT_IO_RATE_CONTROL_INFORMATION {
// 	LONG64 MaxIops;
// 	LONG64 MaxBandwidth;
// 	LONG64 ReservationIops;
// 	PCWSTR VolumeName;
// 	ULONG  BaseIoSize;
// 	ULONG  ControlFlags;
// };
// https://docs.microsoft.com/en-us/windows/desktop/api/jobapi2/ns-jobapi2-jobobject_io_rate_control_information
type _JOBOBJECT_IO_RATE_CONTROL_INFORMATION struct {
	MaxIops         int64
	MaxBandwidth    int64
	ReservationIops int64
	VolumeName      *uint16
	BaseIoSize      uint32
	ControlFlags    uint32
}

// DWORD SetIoRateControlInformationJobObject(
//   HANDLE                                hJob,
//   JOBOBJECT_IO_RATE_CONTROL_INFORMATION *IoRateControlInfo
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/jobapi2/nf-jobapi2-setioratecontrolinformationjobobject
func setIoRateControlInformationJobObject(hJob syscall.Handle, info _JOBOBJECT_IO_RATE_CONTROL_INFORMATION) error {
	ret, _, errno := procSetIoRateControlInformationJobObject.Call(
		uintptr(hJob),
		uintptr(unsafe.Pointer(&info)),
	)
	return testReturnCodeNonZero(ret, errno)
}

// DWORD QueryIoRateControlInformationJobObject(
// 	HANDLE                                hJob,
// 	PCWSTR                                VolumeName,
// 	JOBOBJECT_IO_RATE_CONTROL_INFORMATION **InfoBlocks,
// 	ULONG                                 *InfoBlockCount
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/jobapi2/nf-jobapi2-queryioratecontrolinformationjobobject
func queryIoRateControlInformationJobObject(hJob syscall.Handle, volume string) ([]_JOBOBJECT_IO_RATE_CONTROL_INFORMATION, error) {
	var infos []_JOBOBJECT_IO_RATE_CONTROL_INFORMATION
	var infoBlocks unsafe.Pointer
	var infoBlockCount uint32
	vol := Text(volume).WChars()
	ret, _, errno := procQueryIoRateControlInformationJobObject.Call(
		uintptr(hJob),
		uintptr(unsafe.Pointer(vol)),
		uintptr(unsafe.Pointer(&infoBlocks)),
		uintptr(unsafe.Pointer(&infoBlockCount)),
	)
	if err := testReturnCodeNonZero(ret, errno); err != nil {
		return nil, err
	}
	defer func() {
		LogError(freeMemoryJobObject(uintptr(infoBlocks)), "win32: queryIoRateControlInformationJobObject unable to free InfoBlocks memory. Possible memory leak.")
	}()
	pInfoBlocks := (*[1 << 30]_JOBOBJECT_IO_RATE_CONTROL_INFORMATION)(infoBlocks)[:infoBlockCount:infoBlockCount]
	for _, info := range pInfoBlocks {
		// Copy string out into go memory
		info.VolumeName = Text(UTF16PtrToString(info.VolumeName)).WChars()
		infos = append(infos, info)
	}
	return infos, nil
}

// void FreeMemoryJobObject(
//   _Frees_ptr_ VOID *Buffer
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/jobapi2/nf-jobapi2-freememoryjobobject
func freeMemoryJobObject(buffer uintptr) error {
	_, _, errno := procFreeMemoryJobObject.Call(
		buffer,
	)
	return errnoToError(errno)
}
