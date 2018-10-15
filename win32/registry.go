// +build windows

package win32

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// RegistryKey interfaces with the Windows Registry API
type RegistryKey struct {
	str  string
	hKey HKEY
}

// RegistryKeyPermissions selects the desired permissions
type RegistryKeyPermissions struct {
	Read  bool
	Write bool
}

// OpenRegistryKey opens the registry key with the desired permissions
// The caller is responsible for closing it with the Close function
func OpenRegistryKey(rootKey string, subKey string, perms RegistryKeyPermissions) (*RegistryKey, error) {
	hRootKey, ok := rootKeyHandles[strings.ToUpper(rootKey)]
	if !ok {
		return nil, errors.Errorf("win32: Root key name '%s' not valid", rootKey)
	}
	var access uint32
	if perms.Read {
		access |= _KEY_READ
	}
	if perms.Write {
		access |= _KEY_WRITE
	}
	hKey, err := regOpenKeyExW(hRootKey, subKey, access)
	if err != nil {
		return nil, errors.Wrapf(err, "win32: RegOpenKeyExW failed")
	}
	return &RegistryKey{hKey: hKey, str: fmt.Sprintf("%s\\%s", rootKeyNames[hRootKey], subKey)}, nil
}

// Close releases the registry key resource
func (k *RegistryKey) Close() error {
	if err := regCloseKey(k.hKey); err != nil {
		return errors.Wrapf(err, "win32: regCloseKey failed")
	}
	return nil
}

// ReadValue reads a DWORD value out of the registry key
// It will return an error if the value doesn't exist
func (k *RegistryKey) ReadValue(name string) ([]byte, uint32, error) {
	return readRegValue(k.hKey, name)
}

// String prints the registry key path
func (k *RegistryKey) String() string {
	return k.str
}

// ReadDWORDValue reads a DWORD value out of the registry key
// It will return an error if the value doesn't exist
// or if it is not one of the expected types:
// - REG_DWORD
// - REG_DWORD_BIG_ENDIAN
func (k *RegistryKey) ReadDWORDValue(name string) (uint32, error) {
	kv, kt, err := k.ReadValue(name)
	if err != nil {
		return 0, err
	}
	switch kt {
	case _REG_DWORD:
		return (binary.LittleEndian.Uint32(kv)), nil
	case _REG_DWORD_BIG_ENDIAN:
		return (binary.BigEndian.Uint32(kv)), nil
	}
	return 0, errors.Wrapf(err, "value is not a DWORD")
}
