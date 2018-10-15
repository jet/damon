// +build windows

package main

import (
	"github.com/jet/damon/win32"
)

func addTestUserRights(user string, rights []string) error {
	sid, err := win32.LookupAccountSID("", user)
	if err != nil {
		return err
	}
	privs, err := win32.EnumerateAccountRights(sid)
	if err != nil {
		return err
	}
	privMap := make(map[string]bool)
	for _, p := range privs {
		privMap[p] = true
	}
	var addPrivs []string
	for _, r := range rights {
		if _, ok := privMap[r]; !ok {
			addPrivs = append(addPrivs, r)
			privMap[r] = true
		}
	}
	return win32.AddAccountRights(sid, addPrivs)
}
