// +build windows

package win32

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procCreateRestrictedToken   = advapi32DLL.NewProc("CreateRestrictedToken")
	procGetTokenInformation     = advapi32DLL.NewProc("GetTokenInformation")
	procSetTokenInformation     = advapi32DLL.NewProc("SetTokenInformation")
	procLogonUserW              = advapi32DLL.NewProc("LogonUserW")
	procLookupPrivilegeValue    = advapi32DLL.NewProc("LookupPrivilegeValue")
	procDuplicateTokenEx        = advapi32DLL.NewProc("DuplicateTokenEx")
	procImpersonateLoggedOnUser = advapi32DLL.NewProc("ImpersonateLoggedOnUser")
	procRevertToSelf            = advapi32DLL.NewProc("RevertToSelf")
)

// typedef struct _LUID {
//   DWORD LowPart;
//   LONG  HighPart;
// } LUID, *PLUID;
type _LUID struct {
	LowPart  DWORD
	HighPart LONG
}

// typedef struct _LUID_AND_ATTRIBUTES {
//   LUID  Luid;
//   DWORD Attributes;
// } LUID_AND_ATTRIBUTES, *PLUID_AND_ATTRIBUTES;
type _LUID_AND_ATTRIBUTES struct {
	LUID       _LUID
	Attributes DWORD
}

// BOOL LookupPrivilegeValueW(
//   LPCWSTR lpSystemName,
//   LPCWSTR lpName,
//   PLUID   lpLuid
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/winbase/nf-winbase-lookupprivilegevaluew
func lookupLUID(system *Text, name Text) (*_LUID, error) {
	var luid _LUID
	lpSystemName := system.WChars()
	lpName := name.WChars()
	ret, _, err := procLookupPrivilegeValue.Call(
		uintptr(unsafe.Pointer(lpSystemName)),
		uintptr(unsafe.Pointer(lpName)),
		uintptr(unsafe.Pointer(&luid)),
	)
	if ret == 0 {
		return nil, err
	}
	return &luid, nil
}

// BOOL LogonUserW(
//   LPCWSTR lpszUsername,
//   LPCWSTR lpszDomain,
//   LPCWSTR lpszPassword,
//   DWORD   dwLogonType,
//   DWORD   dwLogonProvider,
//   PHANDLE phToken
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/winbase/nf-winbase-logonuserw

const (
	_LOGON32_LOGON_INTERACTIVE       uint32 = 2
	_LOGON32_LOGON_NETWORK           uint32 = 3
	_LOGON32_LOGON_BATCH             uint32 = 4
	_LOGON32_LOGON_SERVICE           uint32 = 5
	_LOGON32_LOGON_UNLOCK            uint32 = 7
	_LOGON32_LOGON_NETWORK_CLEARTEXT uint32 = 8
	_LOGON32_LOGON_NEW_CREDENTIALS   uint32 = 9
)

const (
	_LOGON32_PROVIDER_DEFAULT uint32 = 0
	_LOGON32_PROVIDER_WINNT35 uint32 = 1
	_LOGON32_PROVIDER_WINNT40 uint32 = 2
	_LOGON32_PROVIDER_WINNT50 uint32 = 3
	_LOGON32_PROVIDER_VIRTUAL uint32 = 4
)

func logonUser(user, domain, password *uint16, logonType uint32, logonProvider uint32) (*syscall.Token, error) {
	var hToken syscall.Token
	ret, _, errno := procLogonUserW.Call(
		uintptr(unsafe.Pointer(user)),
		uintptr(unsafe.Pointer(domain)),
		uintptr(unsafe.Pointer(password)),
		uintptr(logonType),
		uintptr(logonProvider),
		uintptr(unsafe.Pointer(&hToken)),
	)
	if err := testReturnCodeNonZero(ret, errno); err != nil {
		return nil, err
	}
	return &hToken, nil
}

// BOOL WINAPI SetTokenInformation(
//   _In_ HANDLE                  TokenHandle,
//   _In_ TOKEN_INFORMATION_CLASS TokenInformationClass,
//   _In_ LPVOID                  TokenInformation,
//   _In_ DWORD                   TokenInformationLength
// );
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa379591(v=vs.85).aspx
func setTokenInformation(hToken syscall.Token, tokenInformationClass uint32, tokenInformation uintptr, tokenInformationLength uint32) error {
	ret, _, err := procSetTokenInformation.Call(
		uintptr(hToken),
		uintptr(tokenInformationClass),
		tokenInformation,
		uintptr(tokenInformationLength),
	)
	return testReturnCodeNonZero(ret, err)
}

// BOOL GetTokenInformation(
//   HANDLE                  TokenHandle,
//   TOKEN_INFORMATION_CLASS TokenInformationClass,
//   LPVOID                  TokenInformation,
//   DWORD                   TokenInformationLength,
//   PDWORD                  ReturnLength
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/securitybaseapi/nf-securitybaseapi-gettokeninformation
func getTokenInformation(hToken syscall.Token, tokenInformationClass uint32) (unsafe.Pointer, error) {
	n := uint32(4)
	buf := make([]byte, n)
	for {
		ret, _, errno := procGetTokenInformation.Call(
			uintptr(hToken),
			uintptr(tokenInformationClass),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(len(buf)),
			uintptr(unsafe.Pointer(&n)),
		)
		if errno == syscall.ERROR_INSUFFICIENT_BUFFER { // try with bigger buffer
			continue
		}
		if err := testReturnCodeNonZero(ret, errno); err != nil {
			return nil, err
		}
		return unsafe.Pointer(&buf[0]), nil
	}
}

// BOOL DuplicateTokenEx(
//   HANDLE                       hExistingToken,
//   DWORD                        dwDesiredAccess,
//   LPSECURITY_ATTRIBUTES        lpTokenAttributes,
//   SECURITY_IMPERSONATION_LEVEL ImpersonationLevel,
//   TOKEN_TYPE                   TokenType,
//   PHANDLE                      phNewToken
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/securitybaseapi/nf-securitybaseapi-duplicatetokenex
func duplicateTokenEx(hExistingToken syscall.Token, dwDesiredAccess uint32, lpTokenAttributes uintptr, ImpersonationLevel uint32, TokenType uint32) (*syscall.Token, error) {
	var hNewtoken syscall.Token
	ret, _, errno := procDuplicateTokenEx.Call(
		uintptr(hExistingToken),
		uintptr(dwDesiredAccess),
		lpTokenAttributes,
		uintptr(ImpersonationLevel),
		uintptr(TokenType),
		uintptr(unsafe.Pointer(&hNewtoken)),
	)
	if err := testReturnCodeNonZero(ret, errno); err != nil {
		return nil, err
	}
	return &hNewtoken, nil
}

// BOOL ImpersonateLoggedOnUser(
//   HANDLE hToken
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/securitybaseapi/nf-securitybaseapi-impersonateloggedonuser
func impersonateLoggedOnUser(hToken syscall.Token) error {
	ret, _, errno := procImpersonateLoggedOnUser.Call(
		uintptr(hToken),
	)
	return testReturnCodeNonZero(ret, errno)
}

// BOOL RevertToSelf(
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/securitybaseapi/nf-securitybaseapi-reverttoself
func revertToSelf() error {
	ret, _, errno := procRevertToSelf.Call()
	return testReturnCodeNonZero(ret, errno)
}

// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ne-winnt-_security_impersonation_level
const (
	// do not reorder
	_SecurityAnonymous uint32 = iota
	_SecurityIdentification
	_SecurityImpersonation
	_SecurityDelegation
)

// https://docs.microsoft.com/en-us/windows/desktop/api/winnt/ne-winnt-_token_type
const (
	// do not reorder
	_TokenPrimary uint32 = iota + 1
	_TokenImpersonation
)

// BOOL WINAPI CreateRestrictedToken(
//   _In_     HANDLE               ExistingTokenHandle,
//   _In_     DWORD                Flags,
//   _In_     DWORD                DisableSidCount,
//   _In_opt_ PSID_AND_ATTRIBUTES  SidsToDisable,
//   _In_     DWORD                DeletePrivilegeCount,
//   _In_opt_ PLUID_AND_ATTRIBUTES PrivilegesToDelete,
//   _In_     DWORD                RestrictedSidCount,
//   _In_opt_ PSID_AND_ATTRIBUTES  SidsToRestrict,
//   _Out_    PHANDLE              NewTokenHandle
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/securitybaseapi/nf-securitybaseapi-createrestrictedtoken

const (
	_DISABLE_MAX_PRIVILEGE uint32 = 0x1
	_SANDBOX_INERT         uint32 = 0x2
	_LUA_TOKEN             uint32 = 0x4
	_WRITE_RESTRICTED      uint32 = 0x8
)

func createRestrictedToken(hToken syscall.Token, res TokenRestrictions) (*syscall.Token, error) {
	tgr, err := windows.Token(hToken).GetTokenGroups()
	if err != nil {
		return nil, err
	}
	groups := make(map[string]*syscall.SID)
	pGroups := (*[1 << 30]syscall.SIDAndAttributes)(unsafe.Pointer(&tgr.Groups))[:tgr.GroupCount:tgr.GroupCount]
	for i := 0; i < int(tgr.GroupCount); i++ {
		sid := pGroups[i].Sid
		//defer syscall.LocalFree((syscall.Handle)(unsafe.Pointer(sid)))
		account, domain, accType, err := sid.LookupAccount("")
		if err == nil && (accType == syscall.SidTypeGroup || accType == syscall.SidTypeAlias || accType == syscall.SidTypeWellKnownGroup) {
			acct := account
			if domain != "" {
				acct = fmt.Sprintf("%s\\%s", domain, account)
			}
			groups[strings.ToLower(acct)] = sid
		}
	}
	var NewTokenHandle syscall.Token
	var pSidsToDisable *syscall.SIDAndAttributes
	var SidsToDisable []syscall.SIDAndAttributes
	var pPrivilegesToDelete *_LUID_AND_ATTRIBUTES
	var PrivilegesToDelete []_LUID_AND_ATTRIBUTES
	var pSidsToRestrict *syscall.SIDAndAttributes
	var SidsToRestrict []syscall.SIDAndAttributes
	var Flags uint32
	if res.DisableMaxPrivilege {
		Flags |= _DISABLE_MAX_PRIVILEGE
	}
	if res.SandboxInert {
		Flags |= _SANDBOX_INERT
	}
	if res.LUAToken {
		Flags |= _LUA_TOKEN
	}
	if res.WriteRestricted {
		Flags |= _WRITE_RESTRICTED
	}
	for _, s := range res.DisableSIDs {
		if sid, ok := groups[strings.ToLower(s)]; ok {
			defer syscall.LocalFree((syscall.Handle)(uintptr(unsafe.Pointer(sid))))
			SidsToDisable = append(SidsToDisable, syscall.SIDAndAttributes{
				Sid:        sid,
				Attributes: 0,
			})
		}
	}
	if len(SidsToDisable) > 0 {
		pSidsToDisable = &SidsToDisable[0]
	}
	for _, p := range res.DisablePerms {
		luid, err := lookupLUID(nil, Text(p))
		if err != nil {
			return nil, err
		}
		PrivilegesToDelete = append(PrivilegesToDelete, _LUID_AND_ATTRIBUTES{
			LUID:       *luid,
			Attributes: 0,
		})
	}
	if len(PrivilegesToDelete) > 0 {
		pPrivilegesToDelete = &PrivilegesToDelete[0]
	}
	for _, s := range res.RestrictSIDs {
		if sid, ok := groups[strings.ToLower(s)]; ok {
			SidsToRestrict = append(SidsToRestrict, syscall.SIDAndAttributes{
				Sid:        sid,
				Attributes: 0,
			})
		}
	}
	if len(SidsToRestrict) > 0 {
		pSidsToRestrict = &SidsToRestrict[0]
	}
	ret, _, err := procCreateRestrictedToken.Call(
		uintptr(hToken),
		uintptr(Flags),
		uintptr(len(SidsToDisable)),
		uintptr(unsafe.Pointer(pSidsToDisable)),
		uintptr(len(PrivilegesToDelete)),
		uintptr(unsafe.Pointer(pPrivilegesToDelete)),
		uintptr(len(SidsToRestrict)),
		uintptr(unsafe.Pointer(pSidsToRestrict)),
		uintptr(unsafe.Pointer(&NewTokenHandle)),
	)
	return &NewTokenHandle, testReturnCodeNonZero(ret, err)
}
