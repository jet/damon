// +build windows

package win32

import (
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	procCreateEnvironmentBlock  = userenvDLL.NewProc("CreateEnvironmentBlock")
	procDestroyEnvironmentBlock = userenvDLL.NewProc("DestroyEnvironmentBlock")
)

// BOOL CreateEnvironmentBlock(
//   LPVOID *lpEnvironment,
//   HANDLE hToken,
//   BOOL   bInherit
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/userenv/nf-userenv-createenvironmentblock
func createEnvironmentBlock(hToken syscall.Token, inherit bool) (syscall.Handle, error) {
	var lpEnvironment syscall.Handle
	ret, _, errno := procCreateEnvironmentBlock.Call(
		uintptr(unsafe.Pointer(&lpEnvironment)),
		uintptr(hToken),
		uintptr(toBOOL(inherit)),
	)
	if err := testReturnCodeTrue(ret, errno); err != nil {
		return 0, err
	}
	return lpEnvironment, nil
}

// BOOL DestroyEnvironmentBlock(
//   LPVOID lpEnvironment
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/userenv/nf-userenv-destroyenvironmentblock
func destroyEnvironmentBlock(lpEnvironment syscall.Handle) error {
	ret, _, errno := procDestroyEnvironmentBlock.Call(uintptr(lpEnvironment))
	return testReturnCodeTrue(ret, errno)
}

// readEnvironmentBlock reads the environment block into a golang string-array
// The environment block is an array of null-terminated Unicode (UTF-16) strings.
// The list ends with two nulls (\0\0).
func readEnvironmentBlock(lpEnvironment syscall.Handle) []string {
	var envs []string
	var nulls int
	var env []uint16
	for p := lpEnvironment; ; p += 2 {
		u := *(*uint16)(unsafe.Pointer(p))
		if u == 0 {
			nulls++
			if len(env) > 0 {
				envs = append(envs, string(utf16.Decode(env)))
				env = nil
				continue
			}
			if nulls == 2 {
				break
			}
		}
		env = append(env, u)
		nulls = 0
	}
	return envs
}
