// +build windows

package win32

import (
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

var (
	procOpenThread               = kernel32DLL.NewProc("OpenThread")
	procResumeThread             = kernel32DLL.NewProc("ResumeThread")
	procCreateToolhelp32Snapshot = kernel32DLL.NewProc("CreateToolhelp32Snapshot")
	procThread32First            = kernel32DLL.NewProc("Thread32First")
	procThread32Next             = kernel32DLL.NewProc("Thread32Next")
)

// HANDLE OpenThread(
//   DWORD dwDesiredAccess,
//   BOOL  bInheritHandle,
//   DWORD dwThreadId
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/processthreadsapi/nf-processthreadsapi-openthread
func openThread(access uint32, inherit bool, threadID uint32) (*syscall.Handle, error) {
	ret, _, errno := procOpenThread.Call(
		uintptr(access),
		uintptr(toBOOL(inherit)),
		uintptr(threadID),
	)
	hThread := syscall.Handle(ret)
	if ret == NULL {
		return nil, errnoToError(errno)
	}
	return &hThread, nil
}

// DWORD ResumeThread(
//   HANDLE hThread
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/processthreadsapi/nf-processthreadsapi-resumethread
func resumeThread(hThread syscall.Handle) error {
	ret, _, errno := procResumeThread.Call(
		uintptr(hThread),
	)
	if DWORD(ret) == DWORD_MAX {
		return errnoToError(errno)
	}
	return nil
}

// HANDLE CreateToolhelp32Snapshot(
//   DWORD dwFlags,
//   DWORD th32ProcessID
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/tlhelp32/nf-tlhelp32-createtoolhelp32snapshot
func createToolhelp32Snapshot(flags uint32, pid uint32) (*syscall.Handle, error) {
	ret, _, errno := procCreateToolhelp32Snapshot.Call(
		uintptr(flags),
		uintptr(pid),
	)
	hSnapshot := syscall.Handle(ret)
	if hSnapshot == syscall.InvalidHandle {
		return nil, errnoToError(errno)
	}
	return &hSnapshot, nil
}

const (
	_TH32CS_SNAPHEAPLIST uint32 = 0x00000001
	_TH32CS_SNAPPROCESS  uint32 = 0x00000002
	_TH32CS_SNAPTHREAD   uint32 = 0x00000004
	_TH32CS_SNAPMODULE   uint32 = 0x00000008
	_TH32CS_SNAPMODULE32 uint32 = 0x00000010
	_TH32CS_SNAPALL      uint32 = (_TH32CS_SNAPHEAPLIST | _TH32CS_SNAPPROCESS | _TH32CS_SNAPTHREAD | _TH32CS_SNAPMODULE)
	_TH32CS_INHERIT      uint32 = 0x80000000
)

// typedef struct tagTHREADENTRY32 {
//   DWORD dwSize;
//   DWORD cntUsage;
//   DWORD th32ThreadID;
//   DWORD th32OwnerProcessID;
//   LONG  tpBasePri;
//   LONG  tpDeltaPri;
//   DWORD dwFlags;
// } THREADENTRY32;
// https://docs.microsoft.com/en-us/windows/desktop/api/tlhelp32/ns-tlhelp32-tagthreadentry32
type _THREADENTRY32 struct {
	dwSize             uint32
	cntUsage           uint32
	th32ThreadID       uint32
	th32OwnerProcessID uint32
	tpBasePri          int32
	tpDeltaPri         int32
	dwFlags            uint32
}

// Thread Access Rights
// https://docs.microsoft.com/en-us/windows/desktop/ProcThread/thread-security-and-access-rights
const (
	_THREAD_TERMINATE            uint32 = 0x0001
	_THREAD_SUSPEND_RESUME       uint32 = 0x0002
	_THREAD_GET_CONTEXT          uint32 = 0x0008
	_THREAD_SET_CONTEXT          uint32 = 0x0010
	_THREAD_QUERY_INFORMATION    uint32 = 0x0040
	_THREAD_SET_INFORMATION      uint32 = 0x0020
	_THREAD_SET_THREAD_TOKEN     uint32 = 0x0080
	_THREAD_IMPERSONATE          uint32 = 0x0100
	_THREAD_DIRECT_IMPERSONATION uint32 = 0x0200
	// begin_wdm
	_THREAD_SET_LIMITED_INFORMATION   uint32 = 0x0400 // winnt
	_THREAD_QUERY_LIMITED_INFORMATION uint32 = 0x0800 // winnt
	_THREAD_RESUME                    uint32 = 0x1000 // winnt
)

const _THREAD_ALL_ACCESS uint32 = (_STANDARD_RIGHTS_REQUIRED | _SYNCHRONIZE | 0xFFFF)

// BOOL Thread32First(
//   HANDLE          hSnapshot,
//   LPTHREADENTRY32 lpte
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/tlhelp32/nf-tlhelp32-thread32first
func thread32First(hSnapshot syscall.Handle, lpte *_THREADENTRY32) (bool, error) {
	ret, _, errno := procThread32First.Call(
		uintptr(hSnapshot),
		uintptr(unsafe.Pointer(lpte)),
	)
	if errno == syscall.ERROR_NO_MORE_FILES {
		return false, errnoToError(errno)
	}
	return BOOL(ret).boolean(), nil
}

// BOOL Thread32Next(
//   HANDLE          hSnapshot,
//   LPTHREADENTRY32 lpte
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/tlhelp32/nf-tlhelp32-thread32next
func thread32Next(hSnapshot syscall.Handle, lpte *_THREADENTRY32) (bool, error) {
	ret, _, errno := procThread32Next.Call(
		uintptr(hSnapshot),
		uintptr(unsafe.Pointer(lpte)),
	)
	if errno == syscall.ERROR_NO_MORE_FILES {
		return false, errnoToError(errno)
	}
	return BOOL(ret).boolean(), nil
}

// getProcessThreadHandle is a utility function that gets the first thread handle
// of the given process id with the THREAD_RESUME access right
func openProcessMainThreadForResume(pid uint32) (*syscall.Handle, error) {
	phSnapshot, err := createToolhelp32Snapshot(_TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return nil, errors.Wrapf(err, "win32: createToolhelp32Snapshot failed")
	}
	hSnapshot := *phSnapshot
	defer syscall.CloseHandle(hSnapshot)

	var te32 _THREADENTRY32
	te32.dwSize = uint32(unsafe.Sizeof(te32))
	ok, err := thread32First(hSnapshot, &te32)
	for ok && err == nil {
		if te32.th32OwnerProcessID == pid {
			break
		}
		ok, err = thread32Next(hSnapshot, &te32)
	}
	if te32.th32ThreadID != 0 {
		phThread, err := openThread(_THREAD_SUSPEND_RESUME, false, te32.th32ThreadID)
		if err != nil {
			return nil, errors.Wrapf(err, "win32: openThread failed")
		}
		return phThread, nil
	}
	return nil, errors.Errorf("win32: no thread found")
}
