// +build windows

package win32

import (
	"io"
	"syscall"
)

var (
	kernel32DLL = syscall.NewLazyDLL("kernel32.dll")
	advapi32DLL = syscall.NewLazyDLL("advapi32.dll")
	userenvDLL  = syscall.NewLazyDLL("userenv.dll")
	psapiDLL    = syscall.NewLazyDLL("psapi.dll")
	iphlpapiDLL = syscall.NewLazyDLL("iphlpapi.dll")
)

// Types Reference: https://docs.microsoft.com/en-us/windows/desktop/WinProg/windows-data-types
type (
	BOOL          uint32
	BOOLEAN       byte
	BYTE          byte
	DWORD         uint32
	DWORD64       uint64
	HANDLE        uintptr
	HLOCAL        uintptr
	LARGE_INTEGER int64
	LONG          int32
	LPVOID        uintptr
	SIZE_T        uintptr
	UINT          uint32
	ULONG_PTR     uintptr
	ULONGLONG     uint64
	WORD          uint16
)

const (
	NULL uintptr = 0

	ANY_SIZE int = 1

	// Error Codes
	NO_ERROR               uintptr = 0
	ERROR_SUCCESS          uintptr = 0
	ERROR_MORE_DATA        uintptr = 0xea // 234
	ERROR_MR_MID_NOT_FOUND uintptr = 317

	// Booleans
	FALSE BOOL = 0
	TRUE  BOOL = 1

	// Constants
	DWORD_MAX = DWORD(0xFFFFFFFF)
)

// https://docs.microsoft.com/en-us/windows/desktop/SecAuthZ/access-mask-format
const (
	// Generic Access Rights
	// https://docs.microsoft.com/en-us/windows/desktop/SecAuthZ/generic-access-rights

	_GENERIC_READ    uint32 = 0x80000000
	_GENERIC_WRITE   uint32 = 0x40000000
	_GENERIC_EXECUTE uint32 = 0x20000000
	_GENERIC_ALL     uint32 = 0x10000000

	_ACCESS_SYSTEM_SECURITY uint32 = 0x01000000
	_MAXIMUM_ALLOWED        uint32 = 0x02000000

	// Standard Access Rights
	// https://docs.microsoft.com/en-us/windows/desktop/SecAuthZ/standard-access-rights

	_DELETE                   uint32 = 0x00010000
	_READ_CONTROL             uint32 = 0x00020000
	_WRITE_DAC                uint32 = 0x00040000
	_WRITE_OWNER              uint32 = 0x00080000
	_SYNCHRONIZE              uint32 = 0x00100000
	_STANDARD_RIGHTS_REQUIRED        = _DELETE | _READ_CONTROL | _WRITE_DAC | _WRITE_OWNER
	_STANDARD_RIGHTS_EXECUTE         = _READ_CONTROL
	_STANDARD_RIGHTS_READ            = _READ_CONTROL
	_STANDARD_RIGHTS_WRITE           = _READ_CONTROL
	_STANDARD_RIGHTS_ALL             = _DELETE | _READ_CONTROL | _WRITE_DAC | _WRITE_OWNER | _SYNCHRONIZE

	// Object-specific Access Rights mask

	_SPECIFIC_RIGHTS_ALL uint32 = 0x0000ffff
)

func toBOOL(b bool) BOOL {
	if b {
		return TRUE
	}
	return FALSE
}

func (b BOOL) boolean() bool {
	if b == TRUE {
		return true
	}
	return false
}

// testReturnCodeNonZero is a syscall helper function for testing the return code
// for functions that return a handle + error where a zero value is failure
//
// Example:
//
// 		r1, _, errno := procVar.Call(uintptr(x),uintptr(y))
// 		if err := testReturnCodeNonZero(r1, errno); err != nil {
// 		  return nil, err
// 		}
// 		// r1 is valid here
func testReturnCodeNonZero(r1 uintptr, err error) error {
	if r1 == 0 {
		return errnoToError(err)
	}
	return nil
}

// testReturnCodeTrue is a syscall helper function for testing the return code
// for functions that return a handle + error where the return code is a BOOL
// where TRUE is success
//
// Example:
//
// 		r1, _, errno := procVar.Call(uintptr(x),uintptr(y))
// 		if err := testReturnCodeTrue(r1, errno); err != nil {
// 		  return nil, err
// 		}
// 		// r1 is valid here
func testReturnCodeTrue(r1 uintptr, err error) error {
	if !BOOL(r1).boolean() {
		return errnoToError(err)
	}
	return nil
}

func errnoToError(err error) error {
	if errno, ok := err.(syscall.Errno); ok {
		if errno != 0 {
			return errno
		}
		return syscall.EINVAL
	}
	return err
}

func CloseLogErr(c io.Closer, errMsg string) {
	LogError(c.Close(), errMsg)
}

func CloseHandleLogErr(h syscall.Handle, errMsg string) {
	LogError(syscall.CloseHandle(h), errMsg)
}
