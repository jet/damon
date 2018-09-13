// +build windows

package win32

import (
	"syscall"
	"unsafe"
)

var (
	procRegOpenKeyExW    = advapi32DLL.NewProc("RegOpenKeyExW")
	procRegCloseKey      = advapi32DLL.NewProc("RegCloseKey")
	procRegQueryValueExW = advapi32DLL.NewProc("RegQueryValueExW")
)

// LSTATUS RegCloseKey(
//   HKEY hKey
// );
func regCloseKey(hKey HKEY) error {
	ret, _, err := procRegCloseKey.Call(uintptr(hKey))
	if ret != ERROR_SUCCESS {
		return err
	}
	return nil
}

// LSTATUS RegQueryValueExW(
//   HKEY                              hKey,
//   LPCWSTR                           lpValueName,
//   LPDWORD                           lpReserved,
//   LPDWORD                           lpType,
//   __out_data_source(REGISTRY)LPBYTE lpData,
//   LPDWORD                           lpcbData
// );
func readRegValue(hKey HKEY, valueName string) ([]byte, uint32, error) {
	var cbData = uint32(4)
	lpValueName := Text(valueName).WChars()
	var Type uint32
	for {
		var Data = make([]byte, cbData)
		ret, _, err := procRegQueryValueExW.Call(
			uintptr(hKey),
			uintptr(unsafe.Pointer(lpValueName)),
			uintptr(0),
			uintptr(unsafe.Pointer(&Type)),
			uintptr(unsafe.Pointer(&Data[0])),
			uintptr(unsafe.Pointer(&cbData)),
		)
		if err == syscall.ERROR_MORE_DATA {
			continue
		}
		if ret != 0 {
			return nil, 0, err
		}
		return Data[0:cbData], Type, nil
	}
}

// LSTATUS RegOpenKeyExW(
//   HKEY    hKey,
//   LPCWSTR lpSubKey,
//   DWORD   ulOptions,
//   REGSAM  samDesired,
//   PHKEY   phkResult
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/winreg/nf-winreg-regopenkeyexw
func regOpenKeyExW(hRootKey HKEY, subKey string, perms uint32) (HKEY, error) {
	sk, err := syscall.UTF16FromString(subKey)
	if err != nil {
		return 0, err
	}
	var hKeyRes HKEY
	ret, _, err := procRegOpenKeyExW.Call(
		uintptr(hRootKey),
		uintptr(unsafe.Pointer(&sk[0])),
		uintptr(0),
		uintptr(perms),
		uintptr(unsafe.Pointer(&hKeyRes)),
	)
	if ret != ERROR_SUCCESS {
		return 0, err
	}
	return hKeyRes, nil
}

type HKEY uintptr

const (
	_HKEY_CLASSES_ROOT     HKEY = 0x80000000
	_HKEY_CURRENT_USER     HKEY = 0x80000001
	_HKEY_LOCAL_MACHINE    HKEY = 0x80000002
	_HKEY_USERS            HKEY = 0x80000003
	_HKEY_PERFORMANCE_DATA HKEY = 0x80000004
	_HKEY_CURRENT_CONFIG   HKEY = 0x80000005
	_HKEY_DYN_DATA         HKEY = 0x80000006
)

var rootKeyNames = map[HKEY]string{
	_HKEY_CLASSES_ROOT:     "HKEY_CLASSES_ROOT",
	_HKEY_CURRENT_USER:     "HKEY_CURRENT_USER",
	_HKEY_LOCAL_MACHINE:    "HKEY_LOCAL_MACHINE",
	_HKEY_USERS:            "HKEY_USERS",
	_HKEY_PERFORMANCE_DATA: "HKEY_PERFORMANCE_DATA",
	_HKEY_CURRENT_CONFIG:   "HKEY_CURRENT_CONFIG",
	_HKEY_DYN_DATA:         "HKEY_DYN_DATA",
}

var rootKeyShortNames = map[HKEY]string{
	_HKEY_CLASSES_ROOT:     "HKCR",
	_HKEY_CURRENT_USER:     "HKCU",
	_HKEY_LOCAL_MACHINE:    "HKLM",
	_HKEY_USERS:            "HKU",
	_HKEY_PERFORMANCE_DATA: "HKPD",
	_HKEY_CURRENT_CONFIG:   "HKCC",
	_HKEY_DYN_DATA:         "HKDD",
}

var rootKeyHandles = map[string]HKEY{
	// Long Names
	"HKEY_CLASSES_ROOT":     _HKEY_CLASSES_ROOT,
	"HKEY_CURRENT_USER":     _HKEY_CURRENT_USER,
	"HKEY_LOCAL_MACHINE":    _HKEY_LOCAL_MACHINE,
	"HKEY_USERS":            _HKEY_USERS,
	"HKEY_PERFORMANCE_DATA": _HKEY_PERFORMANCE_DATA,
	"HKEY_CURRENT_CONFIG":   _HKEY_CURRENT_CONFIG,
	"HKEY_DYN_DATA":         _HKEY_DYN_DATA,

	// Short Names
	"HKCR": _HKEY_CLASSES_ROOT,
	"HKCU": _HKEY_CURRENT_USER,
	"HKLM": _HKEY_LOCAL_MACHINE,
	"HKU":  _HKEY_USERS,
	"HKPD": _HKEY_PERFORMANCE_DATA,
	"HKCC": _HKEY_CURRENT_CONFIG,
	"HKDD": _HKEY_DYN_DATA,
}

// https://docs.microsoft.com/en-us/windows/desktop/SysInfo/registry-key-security-and-access-rights
const (
	_KEY_ALL_ACCESS         uint32 = 0xf003f
	_KEY_CREATE_LINK        uint32 = 0x0020
	_KEY_CREATE_SUB_KEY     uint32 = 0x0004
	_KEY_ENUMERATE_SUB_KEYS uint32 = 0x0008
	_KEY_EXECUTE                   = _KEY_READ
	_KEY_NOTIFY             uint32 = 0x0010
	_KEY_QUERY_VALUE        uint32 = 0x0001
	_KEY_READ               uint32 = _STANDARD_RIGHTS_READ | _KEY_QUERY_VALUE | _KEY_ENUMERATE_SUB_KEYS | _KEY_NOTIFY
	_KEY_SET_VALUE          uint32 = 0x0002
	_KEY_WOW64_32KEY        uint32 = 0x0200
	_KEY_WOW64_64KEY        uint32 = 0x0100
	_KEY_WRITE                     = _STANDARD_RIGHTS_WRITE | _KEY_SET_VALUE | _KEY_CREATE_SUB_KEY
)

const (
	_REG_NONE                       uint32 = 0 // No value type
	_REG_SZ                         uint32 = 1 // Unicode nul terminated string
	_REG_EXPAND_SZ                  uint32 = 2 // Unicode nul terminated string
	_REG_BINARY                     uint32 = 3 // Free form binary
	_REG_DWORD                      uint32 = 4 // 32-bit number
	_REG_DWORD_LITTLE_ENDIAN        uint32 = _REG_DWORD
	_REG_DWORD_BIG_ENDIAN           uint32 = 5 // 32-bit number
	_REG_LINK                       uint32 = 6 // Symbolic Link = (unicode)
	_REG_MULTI_SZ                   uint32 = 7 // Multiple Unicode strings
	_REG_RESOURCE_LIST              uint32 = 8 // Resource list in the resource map
	_REG_FULL_RESOURCE_DESCRIPTOR   uint32 = 9 // Resource list in the hardware description
	_REG_RESOURCE_REQUIREMENTS_LIST uint32 = 10
	_REG_QWORD                      uint32 = 11 // 64-bit number
	_REG_QWORD_LITTLE_ENDIAN        uint32 = _REG_QWORD
)
