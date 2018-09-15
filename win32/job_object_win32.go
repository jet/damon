// +build windows

package win32

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	procCreateJobObjectW         = kernel32DLL.NewProc("CreateJobObjectW")
	procAssignProcessToJobObject = kernel32DLL.NewProc("AssignProcessToJobObject")
)

// HANDLE WINAPI CreateJobObject(
//   _In_opt_ LPSECURITY_ATTRIBUTES lpJobAttributes,
//   _In_opt_ LPCTSTR               lpName
// );
//
// See https://msdn.microsoft.com/en-us/library/windows/desktop/ms682409(v=vs.85).aspx
func createJobObject(attr *syscall.SecurityAttributes, name string) (syscall.Handle, error) {
	ret, _, err := procCreateJobObjectW.Call(
		uintptr(unsafe.Pointer(attr)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(name))),
	)
	if err != syscall.Errno(0) {
		return 0, err
	}
	return syscall.Handle(ret), nil
}

// BOOL WINAPI AssignProcessToJobObject(
//   _In_ HANDLE hJob,
//   _In_ HANDLE hProcess
// );
// https://msdn.microsoft.com/en-us/f5d7a39f-6afe-4e4a-a802-e7f875ea6e5b
func assignProcessToJobObject(hJob syscall.Handle, hProcess syscall.Handle) error {
	ret, _, err := procAssignProcessToJobObject.Call(
		uintptr(hJob),
		uintptr(hProcess),
	)
	if ret == 0 {
		return err
	}
	return nil
}

// BOOL WINAPI SetInformationJobObject(
//   _In_ HANDLE             hJob,
//   _In_ JOBOBJECTINFOCLASS JobObjectInfoClass,
//   _In_ LPVOID             lpJobObjectInfo,
//   _In_ DWORD              cbJobObjectInfoLength
// );
// https://msdn.microsoft.com/en-us/ms686216
func setInformationJobObject(hJob syscall.Handle, JobObjectInfoClass uint32, lpJobObjectInfo unsafe.Pointer, cbJobObjectInfoLength uint32) error {
	ret, _, err := procSetInformationJobObject.Call(
		uintptr(hJob),
		uintptr(JobObjectInfoClass),
		uintptr(lpJobObjectInfo),
		uintptr(cbJobObjectInfoLength),
	)
	if ret == 0 {
		return err
	}
	return nil
}

func assignJobIOCompletionPort(hJob syscall.Handle, iocp syscall.Handle) error {
	info := _JOBOBJECT_ASSOCIATE_COMPLETION_PORT{
		CompletionKey:  uintptr(hJob),
		CompletionPort: iocp,
	}
	return setInformationJobObject(hJob, _JobObjectAssociateCompletionPortInformation, unsafe.Pointer(&info), uint32(unsafe.Sizeof(info)))
}

// Job Object Messages
// https://docs.microsoft.com/en-us/windows/desktop/api/WinNT/ns-winnt-_jobobject_associate_completion_port
const (
	_JOB_OBJECT_MSG_END_OF_JOB_TIME       = 1
	_JOB_OBJECT_MSG_END_OF_PROCESS_TIME   = 2
	_JOB_OBJECT_MSG_ACTIVE_PROCESS_LIMIT  = 3
	_JOB_OBJECT_MSG_ACTIVE_PROCESS_ZERO   = 4
	_JOB_OBJECT_MSG_NEW_PROCESS           = 6
	_JOB_OBJECT_MSG_EXIT_PROCESS          = 7
	_JOB_OBJECT_MSG_ABNORMAL_EXIT_PROCESS = 8
	_JOB_OBJECT_MSG_PROCESS_MEMORY_LIMIT  = 9
	_JOB_OBJECT_MSG_JOB_MEMORY_LIMIT      = 10
	_JOB_OBJECT_MSG_NOTIFICATION_LIMIT    = 11
	_JOB_OBJECT_MSG_JOB_CYCLE_TIME_LIMIT  = 12
	_JOB_OBJECT_MSG_SILO_TERMINATED       = 13
)

func getQueuedCompletionStatus(hJob syscall.Handle, hCompletion syscall.Handle) (*JobObjectNotification, error) {
	// implementation reference
	// https://github.com/golang/benchmarks/blob/cc0de5f2c23ceaeba742ec694fad6213aba6d252/driver/driver_windows.go
	var code, key uint32
	var o uintptr
	if err := syscall.GetQueuedCompletionStatus(hCompletion, &code, &key, (**syscall.Overlapped)(unsafe.Pointer(&o)), syscall.INFINITE); err != nil {
		return nil, err
	}
	if key != uint32(hJob) {
		return nil, fmt.Errorf("wrong completion key")
	}
	cs := &JobObjectNotification{
		Code:      JobObjectMsgCode(code),
		ProcessID: int(o),
	}
	switch JobObjectMsgCode(code) {
	case _JOB_OBJECT_MSG_END_OF_JOB_TIME:
		break
	case _JOB_OBJECT_MSG_END_OF_PROCESS_TIME:
		cs.ProcessID = int(o)
	case _JOB_OBJECT_MSG_ACTIVE_PROCESS_LIMIT:
		cs.ProcessID = int(o)
	case _JOB_OBJECT_MSG_ACTIVE_PROCESS_ZERO:
		cs.ProcessID = int(o)
	case _JOB_OBJECT_MSG_NEW_PROCESS:
		cs.ProcessID = int(o)
	case _JOB_OBJECT_MSG_EXIT_PROCESS:
		cs.ProcessID = int(o)
	case _JOB_OBJECT_MSG_ABNORMAL_EXIT_PROCESS:
		cs.ProcessID = int(o)
	case _JOB_OBJECT_MSG_PROCESS_MEMORY_LIMIT:
		cs.ProcessID = int(o)
	case _JOB_OBJECT_MSG_JOB_MEMORY_LIMIT:
		break
	case _JOB_OBJECT_MSG_NOTIFICATION_LIMIT:
		cs.ProcessID = int(o)
		st, err := queryJobObjectLimitViolationInformation2(hJob)
		if err != nil {
			return nil, err
		}
		cs.LimitViolationInfo = st.LimitViolationInfo()
	default:
		return nil, fmt.Errorf("unknown job message code: %d", code)
	}
	return cs, nil
}
