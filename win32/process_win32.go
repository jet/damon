// +build windows

package win32

import (
	"syscall"
	"unsafe"
)

var (
	procGenerateConsoleCtrlEvent = kernel32DLL.NewProc("GenerateConsoleCtrlEvent")
	procGetProcessAffinityMask   = kernel32DLL.NewProc("GetProcessAffinityMask")
	procSetProcessAffinityMask   = kernel32DLL.NewProc("SetProcessAffinityMask")
	procOpenProcess              = kernel32DLL.NewProc("OpenProcess")
	procGetProcessMemoryInfo     = psapiDLL.NewProc("GetProcessMemoryInfo")
)

// Process Acecss Rights
// https://docs.microsoft.com/en-us/windows/desktop/ProcThread/process-security-and-access-rights
const (
	// do not reorder
	_PROCESS_TERMINATE uint32 = 1 << iota
	_PROCESS_CREATE_THREAD
	_PROCESS_SET_SESSIONID
	_PROCESS_VM_OPERATION
	_PROCESS_VM_READ
	_PROCESS_VM_WRITE
	_PROCESS_DUP_HANDLE
	_PROCESS_CREATE_PROCESS
	_PROCESS_SET_QUOTA
	_PROCESS_SET_INFORMATION
	_PROCESS_QUERY_INFORMATION
	_PROCESS_SUSPEND_RESUME
	_PROCESS_QUERY_LIMITED_INFORMATION
	_PROCESS_SET_LIMITED_INFORMATION
)

const _PROCESS_ALL_ACCESS uint32 = (_STANDARD_RIGHTS_REQUIRED | _SYNCHRONIZE | 0xFFFF)

// Process Creation Flags
// https://docs.microsoft.com/en-us/windows/desktop/procthread/process-creation-flags
// Scheduling Priorities
// https://docs.microsoft.com/en-us/windows/desktop/procthread/scheduling-priorities
const (
	// do not reorder
	_DEBUG_PROCESS uint32 = 1 << iota
	_DEBUG_ONLY_THIS_PROCESS
	_CREATE_SUSPENDED
	_DETACHED_PROCESS

	_CREATE_NEW_CONSOLE
	_NORMAL_PRIORITY_CLASS
	_IDLE_PRIORITY_CLASS
	_HIGH_PRIORITY_CLASS

	_REALTIME_PRIORITY_CLASS
	_CREATE_NEW_PROCESS_GROUP
	_CREATE_UNICODE_ENVIRONMENT
	_CREATE_SEPARATE_WOW_VDM

	_CREATE_SHARED_WOW_VDM
	_CREATE_FORCEDOS
	_BELOW_NORMAL_PRIORITY_CLASS
	_ABOVE_NORMAL_PRIORITY_CLASS

	_INHERIT_PARENT_AFFINITY
	_INHERIT_CALLER_PRIORITY // Deprecated
	_CREATE_PROTECTED_PROCESS
	_EXTENDED_STARTUPINFO_PRESENT

	_PROCESS_MODE_BACKGROUND_BEGIN
	_PROCESS_MODE_BACKGROUND_END
	_CREATE_SECURE_PROCESS
	_

	_CREATE_BREAKAWAY_FROM_JOB
	_CREATE_PRESERVE_CODE_AUTHZ_LEVEL
	_CREATE_DEFAULT_ERROR_MODE
	_CREATE_NO_WINDOW

	_PROFILE_USER
	_PROFILE_KERNEL
	_PROFILE_SERVER
	_CREATE_IGNORE_SYSTEM_DEFAULT
)

// HANDLE OpenProcess(
// 	DWORD dwDesiredAccess,
// 	BOOL  bInheritHandle,
// 	DWORD dwProcessId
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/processthreadsapi/nf-processthreadsapi-openprocess
func openProcess(access uint32, inherit bool, pid uint32) (*syscall.Handle, error) {
	ret, _, errno := procOpenProcess.Call(
		uintptr(access),
		uintptr(toBOOL(inherit)),
		uintptr(pid),
	)
	if ret == NULL {
		return nil, errnoToError(errno)
	}
	hProcess := syscall.Handle(ret)
	return &hProcess, nil
}

// BOOL WINAPI GenerateConsoleCtrlEvent(
//   _In_ DWORD dwCtrlEvent,
//   _In_ DWORD dwProcessGroupId
// );
// https://docs.microsoft.com/en-us/windows/console/generateconsolectrlevent
func generateConsoleCtrlEvent(dwCtrlEvent uint32, dwProcessGroupId uint32) error {
	ret, _, errno := procGenerateConsoleCtrlEvent.Call(
		uintptr(dwCtrlEvent),
		uintptr(dwProcessGroupId),
	)
	return testReturnCodeNonZero(ret, errno)
}

// BOOL GetProcessAffinityMask(
//   HANDLE     hProcess,
//   PDWORD_PTR lpProcessAffinityMask,
//   PDWORD_PTR lpSystemAffinityMask
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/winbase/nf-winbase-getprocessaffinitymask
func getProcessAffinityMask(hProcess syscall.Handle) (uint32, uint32, error) {
	var pam uint32
	var sam uint32
	ret, _, errno := procGetProcessAffinityMask.Call(
		uintptr(hProcess),
		uintptr(unsafe.Pointer(&pam)),
		uintptr(unsafe.Pointer(&sam)),
	)
	if err := testReturnCodeNonZero(ret, errno); err != nil {
		return 0, 0, err
	}
	return pam, sam, nil
}

// BOOL SetProcessAffinityMask(
//   HANDLE    hProcess,
//   DWORD_PTR dwProcessAffinityMask
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/winbase/nf-winbase-setprocessaffinitymask
func setProcessAffinityMask(hProcess syscall.Handle, sam uint32) error {
	ret, _, errno := procSetProcessAffinityMask.Call(
		uintptr(hProcess),
		uintptr(unsafe.Pointer(&sam)),
	)
	return testReturnCodeNonZero(ret, errno)
}

// typedef struct _PROCESS_MEMORY_COUNTERS_EX {
// 	DWORD  cb;
// 	DWORD  PageFaultCount;
// 	SIZE_T PeakWorkingSetSize;
// 	SIZE_T WorkingSetSize;
// 	SIZE_T QuotaPeakPagedPoolUsage;
// 	SIZE_T QuotaPagedPoolUsage;
// 	SIZE_T QuotaPeakNonPagedPoolUsage;
// 	SIZE_T QuotaNonPagedPoolUsage;
// 	SIZE_T PagefileUsage;
// 	SIZE_T PeakPagefileUsage;
// 	SIZE_T PrivateUsage;
// } PROCESS_MEMORY_COUNTERS_EX;
// https://docs.microsoft.com/en-us/windows/desktop/api/psapi/ns-psapi-_process_memory_counters_ex
type _PROCESS_MEMORY_COUNTERS_EX struct {
	cb                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
	PrivateUsage               uintptr
}

// BOOL GetProcessMemoryInfo(
//   HANDLE                   Process,
//   PPROCESS_MEMORY_COUNTERS ppsmemCounters,
//   DWORD                    cb
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/psapi/nf-psapi-getprocessmemoryinfo
func getProcessMemoryInfo(hProc syscall.Handle) (*_PROCESS_MEMORY_COUNTERS_EX, error) {
	var info _PROCESS_MEMORY_COUNTERS_EX
	info.cb = uint32(unsafe.Sizeof(info))
	ret, _, errno := procGetProcessMemoryInfo.Call(
		uintptr(hProc),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Sizeof(info)),
	)
	if err := testReturnCodeNonZero(ret, errno); err != nil {
		return nil, err
	}
	return &info, nil
}
