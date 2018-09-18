// +build windows

package win32

import (
	"syscall"
	"testing"
	"unsafe"
)

func TestLSA(t *testing.T) {
	login := SetupUserLogin(t)
	pol, err := lsaOpenPolicy("", _POLICY_ALL_ACCESS)
	if err != nil {
		t.Fatal("lsaOpenPolicy", err)
	}
	defer lsaClose(*pol)
	s, err := LookupAccountSID("", login.Username)
	if err != nil {
		t.Fatal("LookupAccountSID", err)
	}
	sid := (*syscall.SID)(unsafe.Pointer(s))
	rights, err := lsaEnumerateAccountRights(*pol, sid)
	if err != nil {
		t.Fatal("lsaEnumerateAccountRights", err)
	}
	for _, r := range rights {
		t.Logf(r)
	}
}
