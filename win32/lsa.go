// +build windows

package win32

import (
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

// RemoveAllAccountRights removes all of the privilege assignments for the given SID in the Local Security Policy
func RemoveAllAccountRights(s *SID) error {
	sid := (*syscall.SID)(unsafe.Pointer(s))
	phPolicy, err := lsaOpenPolicy("", _POLICY_WRITE)
	if err != nil {
		return errors.Wrapf(err, "lsaOpenPolicy")
	}
	defer lsaClose(*phPolicy)
	if err = lsaRemoveAccountRights(*phPolicy, sid, true, nil); err != nil {
		return errors.Wrapf(err, "lsaRemoveAccountRights")
	}
	return nil
}

// RemoveAccountRights removes the given privileges from the given SID in the Local Security Policy
func RemoveAccountRights(s *SID, privs []string) error {
	sid := (*syscall.SID)(unsafe.Pointer(s))
	phPolicy, err := lsaOpenPolicy("", _POLICY_WRITE)
	if err != nil {
		return errors.Wrapf(err, "lsaOpenPolicy")
	}
	defer lsaClose(*phPolicy)
	if err = lsaRemoveAccountRights(*phPolicy, sid, false, privs); err != nil {
		return errors.Wrapf(err, "lsaRemoveAccountRights")
	}
	return nil
}

// AddAccountRights adds the given privileges from the given SID in the Local Security Policy
func AddAccountRights(s *SID, privs []string) error {
	sid := (*syscall.SID)(unsafe.Pointer(s))
	phPolicy, err := lsaOpenPolicy("", _POLICY_WRITE)
	if err != nil {
		return errors.Wrapf(err, "lsaOpenPolicy")
	}
	defer lsaClose(*phPolicy)
	if err = lsaAddAccountRights(*phPolicy, sid, privs); err != nil {
		return errors.Wrapf(err, "lsaAddAccountRights")
	}
	return nil
}

// EnumerateAccountRights returns the list of account privileges assigned to the given SID
func EnumerateAccountRights(s *SID) ([]string, error) {
	sid := (*syscall.SID)(unsafe.Pointer(s))
	phPolicy, err := lsaOpenPolicy("", _POLICY_READ)
	if err != nil {
		return nil, errors.Wrapf(err, "lsaOpenPolicy")
	}
	defer lsaClose(*phPolicy)
	rights, err := lsaEnumerateAccountRights(*phPolicy, sid)
	if err != nil {
		str, _ := sid.String()
		return nil, errors.Wrapf(err, "lsaEnumerateAccountRights(%s)", str)
	}
	return rights, nil
}
