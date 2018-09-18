// +build windows

package win32

import (
	"syscall"
	"unsafe"
)

var (
	procLsaOpenPolicy             = advapi32DLL.NewProc("LsaOpenPolicy")
	procLsaClose                  = advapi32DLL.NewProc("LsaClose")
	procLsaFreeMemory             = advapi32DLL.NewProc("LsaFreeMemory")
	procLsaNtStatusToWinError     = advapi32DLL.NewProc("LsaNtStatusToWinError")
	procLsaAddAccountRights       = advapi32DLL.NewProc("LsaAddAccountRights")
	procLsaEnumerateAccountRights = advapi32DLL.NewProc("LsaEnumerateAccountRights")
	procLsaRemoveAccountRights    = advapi32DLL.NewProc("LsaRemoveAccountRights")
)

// typedef struct _LSA_OBJECT_ATTRIBUTES {
//   ULONG               Length;
//   HANDLE              RootDirectory;
//   PLSA_UNICODE_STRING ObjectName;
//   ULONG               Attributes;
//   PVOID               SecurityDescriptor;
//   PVOID               SecurityQualityOfService;
// } LSA_OBJECT_ATTRIBUTES, *PLSA_OBJECT_ATTRIBUTES;
type _LSA_OBJECT_ATTRIBUTES struct {
	Length                   uint32
	RootDirectory            syscall.Handle
	ObjectName               uintptr
	Attributes               uint32
	SecurityDescriptor       uintptr
	SecurityQualityOfService uintptr
}

// typedef struct _LSA_UNICODE_STRING {
//   USHORT Length;
//   USHORT MaximumLength;
// } LSA_UNICODE_STRING, *PLSA_UNICODE_STRING;
// https://docs.microsoft.com/en-us/windows/desktop/api/lsalookup/ns-lsalookup-_lsa_unicode_string
type _LSA_UNICODE_STRING struct {
	Length        uint16
	MaximumLength uint16
	Buffer        unsafe.Pointer
}

const (
	_POLICY_VIEW_LOCAL_INFORMATION   uint32 = 0x0001
	_POLICY_VIEW_AUDIT_INFORMATION   uint32 = 0x0002
	_POLICY_GET_PRIVATE_INFORMATION  uint32 = 0x0004
	_POLICY_TRUST_ADMIN              uint32 = 0x0008
	_POLICY_CREATE_ACCOUNT           uint32 = 0x0010
	_POLICY_CREATE_SECRET            uint32 = 0x0020
	_POLICY_CREATE_PRIVILEGE         uint32 = 0x0040
	_POLICY_SET_DEFAULT_QUOTA_LIMITS uint32 = 0x0080
	_POLICY_SET_AUDIT_REQUIREMENTS   uint32 = 0x0100
	_POLICY_AUDIT_LOG_ADMIN          uint32 = 0x0200
	_POLICY_SERVER_ADMIN             uint32 = 0x0400
	_POLICY_LOOKUP_NAMES             uint32 = 0x0800
	_POLICY_READ                     uint32 = _STANDARD_RIGHTS_READ | 0x0006
	_POLICY_WRITE                    uint32 = _STANDARD_RIGHTS_WRITE | 0x07F8
	_POLICY_EXECUTE                  uint32 = _STANDARD_RIGHTS_EXECUTE | 0x0801
	_POLICY_ALL_ACCESS               uint32 = _STANDARD_RIGHTS_REQUIRED | 0x0FFF
)

func toLSAUnicodeString(str string) _LSA_UNICODE_STRING {
	wchars, _ := syscall.UTF16FromString(str)
	nc := len(wchars) - 1 // minus 1 to chop off the null termination
	sz := int(unsafe.Sizeof(uint16(0)))
	return _LSA_UNICODE_STRING{
		Length:        uint16(nc * sz),
		MaximumLength: uint16((nc + 1) * sz),
		Buffer:        unsafe.Pointer(&wchars[0]),
	}
}

// NTSTATUS values
// https://msdn.microsoft.com/en-us/library/cc704588.aspx
const (
	_STATUS_SUCCESS      uintptr = 0x00000000
	_STATUS_NO_SUCH_FILE uintptr = 0xC000000F
)

// NTSTATUS LsaOpenPolicy(
// 	PLSA_UNICODE_STRING    SystemName,
// 	PLSA_OBJECT_ATTRIBUTES ObjectAttributes,
// 	ACCESS_MASK            DesiredAccess,
// 	PLSA_HANDLE            PolicyHandle
//   );
// https://docs.microsoft.com/en-us/windows/desktop/api/ntsecapi/nf-ntsecapi-lsaopenpolicy
func lsaOpenPolicy(system string, access uint32) (*syscall.Handle, error) {
	// Docs say this is not used, but the structure needs to be
	// initialized to zero values, and the length must be set to sizeof(_LSA_OBJECT_ATTRIBUTES)
	var pSystemName *_LSA_UNICODE_STRING
	if system != "" {
		lsaStr := toLSAUnicodeString(system)
		pSystemName = &lsaStr
	}
	var attrs _LSA_OBJECT_ATTRIBUTES
	attrs.Length = uint32(unsafe.Sizeof(attrs))
	var hPolicy syscall.Handle
	status, _, _ := procLsaOpenPolicy.Call(
		uintptr(unsafe.Pointer(pSystemName)),
		uintptr(unsafe.Pointer(&attrs)),
		uintptr(access),
		uintptr(unsafe.Pointer(&hPolicy)),
	)
	if status == _STATUS_SUCCESS {
		return &hPolicy, nil
	}
	return nil, lsaNtStatusToWinError(status)
}

// NTSTATUS LsaClose(
//   LSA_HANDLE ObjectHandle
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/ntsecapi/nf-ntsecapi-lsaclose
func lsaClose(hPolicy syscall.Handle) error {
	status, _, _ := procLsaClose.Call(
		uintptr(hPolicy),
	)
	if status == _STATUS_SUCCESS {
		return nil
	}
	return lsaNtStatusToWinError(status)
}

// NTSTATUS LsaEnumerateAccountRights(
//   LSA_HANDLE          PolicyHandle,
//   PSID                AccountSid,
//   PLSA_UNICODE_STRING *UserRights,
//   PULONG              CountOfRights
// );
//https://docs.microsoft.com/en-us/windows/desktop/api/ntsecapi/nf-ntsecapi-lsaenumerateaccountrights
func lsaEnumerateAccountRights(hPolicy syscall.Handle, sid *syscall.SID) ([]string, error) {
	var rights uintptr
	var count uint32
	status, _, _ := procLsaEnumerateAccountRights.Call(
		uintptr(hPolicy),
		uintptr(unsafe.Pointer(sid)),
		uintptr(unsafe.Pointer(&rights)),
		uintptr(unsafe.Pointer(&count)),
	)
	if status != _STATUS_SUCCESS {
		errno := lsaNtStatusToWinError(status)
		if errno == syscall.ERROR_FILE_NOT_FOUND { // user has no rights assigned
			return nil, nil
		}
		return nil, errno
	}
	defer lsaFreeMemory(rights)
	var userRights []string
	rs := (*[1 << 30]_LSA_UNICODE_STRING)(unsafe.Pointer(rights))[:count:count] //nolint
	for _, r := range rs {
		userRights = append(userRights, UTF16PtrToStringN((*uint16)(r.Buffer), int(r.Length/2)))
	}
	return userRights, nil
}

// NTSTATUS LsaAddAccountRights(
// 	LSA_HANDLE          PolicyHandle,
// 	PSID                AccountSid,
// 	PLSA_UNICODE_STRING UserRights,
// 	ULONG               CountOfRights
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/ntsecapi/nf-ntsecapi-lsaaddaccountrights
func lsaAddAccountRights(hPolicy syscall.Handle, sid *syscall.SID, rights []string) error {
	var lsaRights []_LSA_UNICODE_STRING
	for _, r := range rights {
		lsaRights = append(lsaRights, toLSAUnicodeString(r))
	}
	status, _, _ := procLsaAddAccountRights.Call(
		uintptr(hPolicy),
		uintptr(unsafe.Pointer(sid)),
		uintptr(unsafe.Pointer(&lsaRights[0])),
		uintptr(len(rights)),
	)
	if status != _STATUS_SUCCESS {
		return lsaNtStatusToWinError(status)
	}
	return nil
}

// NTSTATUS LsaRemoveAccountRights(
//   LSA_HANDLE          PolicyHandle,
//   PSID                AccountSid,
//   BOOLEAN             AllRights,
//   PLSA_UNICODE_STRING UserRights,
//   ULONG               CountOfRights
// );
//https://docs.microsoft.com/en-us/windows/desktop/api/ntsecapi/nf-ntsecapi-lsaremoveaccountrights
func lsaRemoveAccountRights(hPolicy syscall.Handle, sid *syscall.SID, removeAll bool, rights []string) error {
	var lsaRights []_LSA_UNICODE_STRING
	if !removeAll {
		for _, r := range rights {
			lsaRights = append(lsaRights, toLSAUnicodeString(r))
		}
	}
	status, _, _ := procLsaRemoveAccountRights.Call(
		uintptr(hPolicy),
		uintptr(unsafe.Pointer(sid)),
		uintptr(toBOOL(removeAll)),
		uintptr(unsafe.Pointer(&lsaRights[0])),
		uintptr(len(lsaRights)),
	)
	if status != _STATUS_SUCCESS {
		return lsaNtStatusToWinError(status)
	}
	return nil
}

// ULONG LsaNtStatusToWinError(
//   NTSTATUS Status
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/ntsecapi/nf-ntsecapi-lsantstatustowinerror
func lsaNtStatusToWinError(status uintptr) error {
	ret, _, _ := procLsaNtStatusToWinError.Call(status)
	if ret == ERROR_MR_MID_NOT_FOUND {
		return syscall.EINVAL
	}
	return syscall.Errno(ret)
}

// NTSTATUS LsaFreeMemory(
// 	PVOID Buffer
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/ntsecapi/nf-ntsecapi-lsafreememory
func lsaFreeMemory(buf uintptr) error {
	status, _, _ := procLsaFreeMemory.Call(buf)
	if status == _STATUS_SUCCESS {
		return nil
	}
	return lsaNtStatusToWinError(status)
}
