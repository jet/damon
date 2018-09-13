// +build windows

package win32

import (
	"syscall"

	"github.com/pkg/errors"
)

type Token struct {
	hToken syscall.Token
}

type TokenType uint32

const (
	TokenTypePrimary       TokenType = 1
	TokenTypeImpersonation TokenType = 2
)

// TokenRestrictions are parameters to CreateRestrictedToken
type TokenRestrictions struct {
	DisableMaxPrivilege bool
	SandboxInert        bool
	LUAToken            bool
	WriteRestricted     bool
	DisableSIDs         []string
	DisablePerms        []string
	RestrictSIDs        []string
}

// CreateRestrictedToken creates a restricted token from an existing token
func (t *Token) CreateRestrictedToken(res TokenRestrictions) (*Token, error) {
	phResToken, err := createRestrictedToken(t.hToken, res)
	if err != nil {
		return nil, errors.Wrapf(err, "win32: createRestrictedToken failed")
	}
	return &Token{hToken: *phResToken}, nil
}

// TokenType gets the token type value
func (t *Token) TokenType() (TokenType, error) {
	tt, err := getTokenInformation(t.hToken, syscall.TokenType)
	if err != nil {
		return 0, errors.Wrapf(err, "win32: getTokenInformation failed")
	}
	return *(*TokenType)(tt), nil
}

func (t *Token) Environment(inherit bool) ([]string, error) {
	lpEnvironment, err := createEnvironmentBlock(t.hToken, inherit)
	if err != nil {
		return nil, err
	}
	defer func() {
		LogError(destroyEnvironmentBlock(lpEnvironment), "win32.DestroyEnvironmentBlock failed ")
	}()
	return readEnvironmentBlock(lpEnvironment), nil
}

// RunAs runs the given function in the context of this token
func (t *Token) RunAs(fn func()) error {
	if err := impersonateLoggedOnUser(t.hToken); err != nil {
		return err
	}
	defer func() {
		LogError(revertToSelf(), "win32.RevertToSelf failed")
	}()
	fn()
	return nil
}

// Close the token handle
func (t *Token) Close() error {
	return t.hToken.Close()
}

// CurrentProcessToken returns the current process token
func CurrentProcessToken() (*Token, error) {
	hProc, err := syscall.GetCurrentProcess()
	if err != nil {
		return nil, errors.Wrapf(err, "win32: GetCurrentProcess failed")
	}
	defer CloseHandleLogErr(syscall.Handle(hProc), "win32: failed to close process handle")
	var hToken syscall.Token
	if err = syscall.OpenProcessToken(hProc,
		syscall.TOKEN_DUPLICATE|syscall.TOKEN_ADJUST_DEFAULT|syscall.TOKEN_QUERY|syscall.TOKEN_ASSIGN_PRIMARY,
		&hToken); err != nil {
		return nil, errors.Wrapf(err, "win32: OpenProcessToken failed")
	}
	return &Token{
		hToken: hToken,
	}, nil
}

// UserLogin is the user's login credentials for making a user access token
type UserLogin struct {
	Domain   string
	Username string
	Password Password
}

// Password abstracts how the UTF-16 / ANSI representation of the password is stored
// in order to support secure strings.
type Password interface {
	PasswordW() *uint16
	PasswordA() *uint8
}

// UnsafePasswordString is an implementation of Password that uses a golang string
// which is not secured in any way
type UnsafePasswordString string

// PasswordW returns the UTF-16 encoding of the password
func (s UnsafePasswordString) PasswordW() *uint16 {
	return Text(s).WChars()
}

// PasswordA returns the ANSI encoding of the password
func (s UnsafePasswordString) PasswordA() *uint8 {
	return Text(s).Chars()
}

// CreateBatchUserToken creates a new primary token using Service Login
// the user must have the "Log on as a batch job" right as set by Group Policy
// This is located in the Group Policy Editor (gpedit.msc) under:
// Windows Settings >> Security Settings >> Local Policies >> User Rights Assignment
func CreateBatchUserToken(login UserLogin) (*Token, error) {
	username := Text(login.Username).WChars()
	domain := Text(login.Domain).WChars()
	password := login.Password.PasswordW()
	phToken, err := logonUser(username, domain, password, _LOGON32_LOGON_BATCH, _LOGON32_PROVIDER_DEFAULT)
	if err != nil {
		return nil, errors.Wrapf(err, "win32: logonUser as Batch failure")
	}
	defer CloseLogErr(phToken, "win32: unable to close LogonUser token handle")
	phNewToken, err := duplicateTokenEx(*phToken, _GENERIC_ALL, NULL, _SecurityImpersonation, _TokenPrimary)
	if err != nil {
		return nil, errors.Wrapf(err, "win32: duplicateTokenEx failed")
	}
	return &Token{hToken: *phNewToken}, nil
}
