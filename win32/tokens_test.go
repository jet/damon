// +build windows

package win32

import (
	"os"
	"testing"
)

func SetupUserLogin(t *testing.T) UserLogin {
	t.Helper()
	login := UserLogin{
		Domain:   os.Getenv("TEST_WIN32_USER_DOMAIN"),
		Username: os.Getenv("TEST_WIN32_USER_NAME"),
		Password: UnsafePasswordString(os.Getenv("TEST_WIN32_USER_PASSWORD")),
	}
	if login.Username == "" {
		t.Skip("TEST_WIN32_USER_NAME is empty")
	}
	return login
}

func TestCurrentProcessToken(t *testing.T) {
	token, err := CurrentProcessToken()
	if err != nil {
		t.Fatal(err)
	}
	defer token.Close()
	tt, err := token.TokenType()
	if err != nil {
		t.Fatal("TokenType", err)
	}
	if tt != TokenTypePrimary {
		t.Error("TokenType is Impersonation; should be TokenTypePrimary")
	}
}

func TestCreateBatchUserTokenBadPassword(t *testing.T) {
	login := SetupUserLogin(t)
	login.Password = UnsafePasswordString("___BAD_PASSWORD___")
	token, err := CreateBatchUserToken(login)
	if err == nil {
		defer token.Close()
		t.Fatal("CreateBatchUserToken: unexpected success")
	}
	t.Log(err)
}

func TestCreateBatchUserToken(t *testing.T) {
	login := SetupUserLogin(t)
	token, err := CreateBatchUserToken(login)
	if err != nil {
		t.Fatal("CreateBatchUserToken", err)
	}
	defer token.Close()
	tt, err := token.TokenType()
	if err != nil {
		t.Fatal("token.TokenType", err)
	}
	if tt != TokenTypePrimary {
		t.Error("token.TokenType is Impersonation; should be TokenTypePrimary")
	}
	envs, err := token.Environment(false)
	if err != nil {
		t.Error("token.Environment error", err)
	}
	for _, env := range envs {
		t.Logf(env)
	}
	restricted, err := token.CreateRestrictedToken(TokenRestrictions{
		DisableMaxPrivilege: true,
		DisableSIDs: []string{
			"BUILTIN\\Administrators",
			"BUILTIN\\Backup Operators",
			"BUILTIN\\Performance Log Users",
		},
	})
	if err != nil {
		t.Fatal("CreateRestrictedToken", err)
	}
	defer restricted.Close()
	rtt, err := restricted.TokenType()
	if err != nil {
		t.Fatal("restricted.TokenType", err)
	}
	if rtt != TokenTypePrimary {
		t.Error("restricted.TokenType is Impersonation; should be TokenTypePrimary")
	}
}
