// +build windows

package win32

import (
	"syscall"
	"unicode/utf16"
	"unsafe"
)

type Text string

func (t Text) String() string {
	return string(t)
}

func (t Text) Chars() *uint8 {
	if t == "" {
		return nil
	}
	return StringToCharPtr(string(t))
}

func (t Text) WChars() *uint16 {
	if t == "" {
		return nil
	}
	bs, _ := syscall.UTF16FromString(string(t))
	return &bs[0]
}

// CharPtrToString converts a null-terminated string
// to a Go string. The C-String should be an ASCII-encoded string.
// This method does not support Wide-Characters.
// For Wide-characters, use UTF16PtrToString
func CharPtrToString(cstr *uint8) string {
	if cstr == nil {
		return ""
	}
	var chars []byte
	for p := uintptr(unsafe.Pointer(cstr)); ; p++ {
		//nolint
		ch := *(*uint8)(unsafe.Pointer(p))
		if ch == 0 {
			return string(chars)
		}
		chars = append(chars, ch)
	}
}

// StringToCharPtr converts a go string into a null-terminated string
// The string should be an ASCII-encoded, not UTF-8.
func StringToCharPtr(str string) *uint8 {
	if str == "" {
		n := []uint8{0}
		return &n[0]
	}
	chars := append([]byte(str), 0) // null terminated
	return &chars[0]
}

// UTF16PtrToString converts a null-terimanted UTF-16 encoded C-String
// into a Go string. This method supports only wide-character
// strings in UTF-16; not UTF-8.
// For ASCII strings, use CharPtrToString.
func UTF16PtrToString(wstr *uint16) string {
	if wstr != nil {
		us := make([]uint16, 0, 256)
		for p := uintptr(unsafe.Pointer(wstr)); ; p += 2 {
			//nolint
			u := *(*uint16)(unsafe.Pointer(p))
			if u == 0 {
				return string(utf16.Decode(us))
			}
			us = append(us, u)
		}
	}
	return ""
}

// UTF16PtrToStringN converts a UTF-16 encoded C-String
// into a Go string. The n specifies the length of the string.
// This function supports only wide-character strings in UTF-16; not UTF-8.
func UTF16PtrToStringN(wstr *uint16, n int) string {
	if wstr != nil {
		us := make([]uint16, 0, n)
		i := 0
		for p := uintptr(unsafe.Pointer(wstr)); ; p += 2 {
			//nolint
			u := *(*uint16)(unsafe.Pointer(p))
			us = append(us, u)
			i++
			if i > n {
				return string(utf16.Decode(us))
			}
		}
	}
	return ""
}
