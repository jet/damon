// +build windows

package win32

import (
	"testing"
)

func TestLookupAccountSID(t *testing.T) {
	adminSID, err := LookupAccountSID("", "BUILTIN\\Administrators")
	if err != nil {
		t.Fatal("LookupAccountSID('BUILTIN\\Administrators')", err)
	}
	actual := adminSID.String()
	expected := string(SIDAdministrators)
	if actual != expected {
		t.Fatalf("expected '%s', actual '%s", expected, actual)
	}
}

func TestLookupAccountSIDBadName(t *testing.T) {
	if _, err := LookupAccountSID("", "DOESNOT\\EXIST"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSIDs(t *testing.T) {
	sidStrings := []StringSID{
		SIDAdministrators,
		SIDUsers,
		SIDGuests,
		SIDPowerUsers,
		SIDAccountOperators,
		SIDServerOperators,
		SIDPrintOperators,
		SIDBackupOperators,
		SIDReplicators,
		SIDNTLMAuthentication,
		SIDSChannelAuthentication,
		SIDDigestAuthentication,
		SIDAllServices,
		SIDNTVirtualMachines,
	}
	for _, s := range sidStrings {
		t.Run(string(s), func(t *testing.T) {
			sid, err := s.ConvertToSID()
			if err != nil {
				t.Error("ConvertToSID failed", err)
			}
			sidCopy, err := sid.Copy()
			if err != nil {
				t.Error("sid.Copy failed", err)
			}
			if sidCopy == sid {
				t.Error("expected sidCopy != sid")
			}
			str, err := sid.StringErr()
			if err != nil {
				t.Error("sid.StringErr failed", err)
			}
			if str != string(s) {
				t.Errorf("sid.StringErr(): expected '%s', actual '%s", string(s), str)
			}
			str2 := sidCopy.String()
			if str2 != string(s) {
				t.Errorf("sidCopy.String(): expected '%s', actual '%s", string(s), str2)
			}
		})
	}
}
