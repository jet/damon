// +build windows

package win32

import (
	"syscall"
	"unsafe"
)

// https://support.microsoft.com/en-us/help/243330/well-known-security-identifiers-in-windows-operating-systems

const (
	// SIDAdministrators is a built-in group. After the initial installation of the operating system, the only member of the group is the Administrator account. When a computer joins a domain, the Domain Admins group is added to the Administrators group. When a server becomes a domain controller, the Enterprise Admins group also is added to the Administrators group.
	SIDAdministrators = StringSID("S-1-5-32-544")

	// SIDUsers is a built-in group. After the initial installation of the operating system, the only member is the Authenticated Users group. When a computer joins a domain, the Domain Users group is added to the Users group on the computer.
	SIDUsers = StringSID("S-1-5-32-545")

	// SIDGuests is a built-in group. By default, the only member is the Guest account. The Guests group allows occasional or one-time users to log on with limited privileges to a computer's built-in Guest account.
	SIDGuests = StringSID("S-1-5-32-546")

	// SIDPowerUsers is a built-in group. By default, the group has no members. Power users can create local users and groups; modify and delete accounts that they have created; and remove users from the Power Users, Users, and Guests groups. Power users also can install programs; create, manage, and delete local printers; and create and delete file shares.
	SIDPowerUsers = StringSID("S-1-5-32-547")

	// SIDAccountOperators is a built-in group that exists only on domain controllers. By default, the group has no members. By default, Account Operators have permission to create, modify, and delete accounts for users, groups, and computers in all containers and organizational units of Active Directory except the Builtin container and the Domain Controllers OU. Account Operators do not have permission to modify the Administrators and Domain Admins groups, nor do they have permission to modify the accounts for members of those groups.
	SIDAccountOperators = StringSID("S-1-5-32-548")

	// SIDServerOperators is a built-in group that exists only on domain controllers. By default, the group has no members. Server Operators can log on to a server interactively; create and delete network shares; start and stop services; back up and restore files; format the hard disk of the computer; and shut down the computer.
	SIDServerOperators = StringSID("S-1-5-32-549")

	// SIDPrint Operators is a built-in group that exists only on domain controllers. By default, the only member is the Domain Users group. Print Operators can manage printers and document queues.
	SIDPrintOperators = StringSID("S-1-5-32-550")

	// SIDBackup Operators is a built-in group. By default, the group has no members. Backup Operators can back up and restore all files on a computer, regardless of the permissions that protect those files. Backup Operators also can log on to the computer and shut it down.
	SIDBackupOperators = StringSID("S-1-5-32-551")

	// SIDReplicators is a built-in group that is used by the File Replication service on domain controllers. By default, the group has no members. Do not add users to this group.
	SIDReplicators = StringSID("S-1-5-32-552")

	// SIDNTLMAuthentication is a SID that is used when the NTLM authentication package authenticated the client
	SIDNTLMAuthentication = StringSID("S-1-5-64-10")

	// SIDSChannelAuthentication is a SID that is used when the SChannel authentication package authenticated the client.
	SIDSChannelAuthentication = StringSID("S-1-5-64-14")

	// SIDDigestAuthentication is a SID that is used when the Digest authentication package authenticated the client.
	SIDDigestAuthentication = StringSID("S-1-5-64-21")

	// SIDAllServices is a group that includes all service processes that are configured on the system. Membership is controlled by the operating system.
	SIDAllServices = StringSID("S-1-5-80-0")

	// SIDNTVirtualMachines is a built-in group. The group is created when the Hyper-V role is installed. Membership in the group is maintained by the Hyper-V Management Service (VMMS). This group requires the "Create Symbolic Links" right (SeCreateSymbolicLinkPrivilege), and also the "Log on as a Service" right (SeServiceLogonRight).
	SIDNTVirtualMachines = StringSID("S-1-5-83-0")
)

const (
	// SIDUntrustedMandatoryLevel is an untrusted integrity level. Note Added in Windows Vista and Windows Server 2008
	SIDUntrustedMandatoryLevel = StringSID("S-1-16-0")

	// SIDLowMandatoryLevel is a low integrity level.
	SIDLowMandatoryLevel = StringSID("S-1-16-4096")

	// SIDMediumMandatoryLevel is a medium integrity level.
	SIDMediumMandatoryLevel = StringSID("S-1-16-8192")

	// SIDMediumPlusMandatoryLevel is a medium plus integrity level.
	SIDMediumPlusMandatoryLevel = StringSID("S-1-16-8448")

	// SIDHighMandatoryLevel is a high integrity level.
	SIDHighMandatoryLevel = StringSID("S-1-16-12288")

	// SIDSystemMandatoryLevel is a system integrity level.
	SIDSystemMandatoryLevel = StringSID("S-1-16-16384")

	// SIDProtectedProcessMandatoryLevel is a protected-process integrity level.
	SIDProtectedProcessMandatoryLevel = StringSID("S-1-16-20480")

	// SIDSecureProcessMandatoryLevel is a secure process integrity level.
	SIDSecureProcessMandatoryLevel = StringSID("S-1-16-28672")
)

// StringSID is a string representation of a SID
type StringSID string

// SID is a windows security identifier
type SID struct{}

// Copy the SID
func (s *SID) Copy() (*SID, error) {
	sid := (*syscall.SID)(unsafe.Pointer(s))
	nsid, err := sid.Copy()
	if err != nil {
		return nil, err
	}
	r := (*SID)(unsafe.Pointer(nsid))
	return r, nil
}

// StringErr gets the string representation of the SID
// Returns an error if the syscall fails
func (s *SID) StringErr() (string, error) {
	sid := (*syscall.SID)(unsafe.Pointer(s))
	return sid.String()
}

// String gets the string representation of the SID
// If there is an error getting the string, then it will return an empty string
func (s *SID) String() string {
	str, err := s.StringErr()
	if err != nil {
		return ""
	}
	return str
}

// ConvertToSID converts this string SID into a SID
func (s StringSID) ConvertToSID() (*SID, error) {
	var sid *syscall.SID
	err := syscall.ConvertStringSidToSid(Text(s).WChars(), &sid)
	r := (*SID)(unsafe.Pointer(sid))
	return r, err
}

// LookupAccountSID looks up a SID given a system name and account name
// this system name is optional.LookupAccountSID
func LookupAccountSID(system string, name string) (*SID, error) {
	sid, _, _, err := syscall.LookupSID(system, name)
	if err != nil {
		return nil, err
	}
	r := (*SID)(unsafe.Pointer(sid))
	return r, nil
}
