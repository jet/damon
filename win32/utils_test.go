// +build windows

package win32

import (
	"os"
	"path/filepath"
	"testing"
)

func SetupTestExe(t *testing.T) string {
	if exe := os.Getenv("TEST_EXE_PATH"); exe != "" {
		stat, err := os.Stat(exe)
		if err != nil {
			t.Skipf("unable to stat test.exe: %v", err)
		}
		if stat.IsDir() {
			t.Skipf("test.exe is a directory")
		}
		abs, err := filepath.Abs(exe)
		if err != nil {
			t.Skipf("unable to get absolute path of test.exe: %v", err)
		}
		return abs
	}
	t.Skip("TEST_EXE_PATH not set")
	return ""
}

func LogTestError(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
	}
}
