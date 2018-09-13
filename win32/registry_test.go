// +build windows

package win32

import "testing"

func TestRegistryKeyRead(t *testing.T) {
	val := "CurrentMajorVersionNumber"
	key, err := OpenRegistryKey("HKEY_LOCAL_MACHINE", `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, RegistryKeyPermissions{Read: true})
	if err != nil {
		t.Fatal("OpenRegistryKey", err)
	}
	defer key.Close()
	dw, err := key.ReadDWORDValue("CurrentMajorVersionNumber")
	if err != nil {
		t.Fatal("ReadDWORDValue('CurrentMajorVersionNumber')", err)
	}
	t.Logf("%v['%s'] = 0x%x", key, val, dw)
}

func TestRegistryKeyReadBadType(t *testing.T) {
	val := "SystemRoot"
	key, err := OpenRegistryKey("HKEY_LOCAL_MACHINE", `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, RegistryKeyPermissions{Read: true})
	if err != nil {
		t.Fatal("OpenRegistryKey", err)
	}
	defer key.Close()
	if _, err = key.ReadDWORDValue(val); err == nil {
		t.Fatal("expected ReadDWORDValue to fail")
	}
}

func TestRegistryKeyBadKey(t *testing.T) {
	key, err := OpenRegistryKey("HKEY_LOCAL_MACHINE", `DOES_NOT_EXIST`, RegistryKeyPermissions{Read: true})
	if err == nil {
		key.Close()
		t.Fatal("expected error")
	}
}

func TestRegistryKeyBadRootKey(t *testing.T) {
	key, err := OpenRegistryKey("HKEY_ROOT_DOES_NOT_EXIST", `DOES_NOT_EXIST`, RegistryKeyPermissions{Read: true})
	if err == nil {
		key.Close()
		t.Fatal("expected error")
	}
}
