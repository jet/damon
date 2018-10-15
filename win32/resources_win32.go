// +build windows

package win32

import (
	"syscall"
	"unsafe"
)

var (
	procGetSystemInfo        = kernel32DLL.NewProc("GetSystemInfo")
	procGlobalMemoryStatusEx = kernel32DLL.NewProc("GlobalMemoryStatusEx")
)

// typedef struct _SYSTEM_INFO {
//   union {
//     DWORD  dwOemId;
//     struct {
//       WORD wProcessorArchitecture;
//       WORD wReserved;
//     };
//   };
//   DWORD     dwPageSize;
//   LPVOID    lpMinimumApplicationAddress;
//   LPVOID    lpMaximumApplicationAddress;
//   DWORD_PTR dwActiveProcessorMask;
//   DWORD     dwNumberOfProcessors;
//   DWORD     dwProcessorType;
//   DWORD     dwAllocationGranularity;
//   WORD      wProcessorLevel;
//   WORD      wProcessorRevision;
// } SYSTEM_INFO;
// https://msdn.microsoft.com/en-us/library/ms724958(v=vs.85).aspx
type _SYSTEM_INFO struct {
	dwOemId                     DWORD
	dwPageSize                  DWORD
	lpMinimumApplicationAddress uintptr
	lpMaximumApplicationAddress uintptr
	dwActiveProcessorMask       uintptr
	dwNumberOfProcessors        DWORD
	dwProcessorType             DWORD
	dwAllocationGranularity     DWORD
	wProcessorLevel             WORD
	wProcessorRevision          WORD
}

// void WINAPI GetSystemInfo(
//   _Out_ LPSYSTEM_INFO lpSystemInfo
// );
// https://msdn.microsoft.com/en-us/library/ms724381(VS.85).aspx
func getSystemInfo() (_SYSTEM_INFO, error) {
	var si _SYSTEM_INFO
	_, _, err := procGetSystemInfo.Call(
		uintptr(unsafe.Pointer(&si)),
	)
	if err != syscall.Errno(0) {
		return si, err
	}
	return si, nil
}

// typedef struct _MEMORYSTATUSEX {
// 	DWORD     dwLength;
// 	DWORD     dwMemoryLoad;
// 	DWORDLONG ullTotalPhys;
// 	DWORDLONG ullAvailPhys;
// 	DWORDLONG ullTotalPageFile;
// 	DWORDLONG ullAvailPageFile;
// 	DWORDLONG ullTotalVirtual;
// 	DWORDLONG ullAvailVirtual;
// 	DWORDLONG ullAvailExtendedVirtual;
// } MEMORYSTATUSEX, *LPMEMORYSTATUSEX;
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa366770(v=vs.85).aspx
type _MEMORYSTATUSEX struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

// BOOL WINAPI GlobalMemoryStatusEx(
// 	_Inout_ LPMEMORYSTATUSEX lpBuffer
// );
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa366589(v=vs.85).aspx
func globalMemoryStatusEx() (*_MEMORYSTATUSEX, error) {
	var Buffer _MEMORYSTATUSEX
	Buffer.dwLength = uint32(unsafe.Sizeof(Buffer))
	ret, _, errno := procGlobalMemoryStatusEx.Call(
		uintptr(unsafe.Pointer(&Buffer)),
	)
	if err := testReturnCodeNonZero(ret, errno); err != nil {
		return nil, err
	}
	return &Buffer, nil
}
