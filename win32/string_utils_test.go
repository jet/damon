// +build windows

package win32

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"
)

type StringTestFunc func(t *testing.T, str string)

func RunTestStrings(t *testing.T, fn StringTestFunc) {
	t.Helper()
	fd, err := os.Open("./testdata/strings.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer fd.Close()
	br := bufio.NewReader(fd)
	var eof bool
	var lineno int
	for !eof {
		lineno++
		line, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				eof = true
			} else {
				t.Error("error reading test data", err)
			}
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !utf8.ValidString(line) {
			// not valid unicode this line
			continue
		}
		t.Run(fmt.Sprintf("testdata/strings.txt:%d", lineno), func(t *testing.T) {
			t.Helper()
			fn(t, line)
		})
	}
}

func TestTextCharPtrToString(t *testing.T) {
	RunTestStrings(t, func(t *testing.T, str string) {
		text := Text(str)
		if cstr := CharPtrToString(text.Chars()); cstr != str {
			t.Errorf("CharPtrToString(text.Chars()) != str: %s != %s", text, str)
		}
	})
}
func TestUTF16PtrToString(t *testing.T) {
	RunTestStrings(t, func(t *testing.T, str string) {
		utf16Str := append(utf16.Encode([]rune(str)), 0) // convert to UTF-16, NULL-terminated string
		if wstr := UTF16PtrToString(&utf16Str[0]); wstr != str {
			t.Errorf("UTF16PtrToString(&utf16Str[0]) != str: %s != %s", wstr, str)
		}
	})
}
