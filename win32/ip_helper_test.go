// +build windows

package win32

import "testing"

func TestTCPTableeOwnerPID(t *testing.T) {
	table, err := GetTCPTableIP4OwnerPID(true, TCPTableAll)
	if err != nil {
		t.Fatal(err)
	}
	for i, row := range table {
		if row.State != TcpListen {
			t.Logf("%d: %d: %v:%d => %v:%d %s", i, row.PID, row.LocalAddress, row.LocalPort, row.RemoteAddress, row.RemotePort, row.State)
		} else {
			t.Logf("%d: %d: %v:%d %s", i, row.PID, row.LocalAddress, row.LocalPort, row.State)
		}
	}
}

func TestTCP6TableOwnerPID(t *testing.T) {
	table, err := GetTCPTableIP6OwnerPID(true, TCPTableAll)
	if err != nil {
		t.Fatal(err)
	}
	for i, row := range table {
		if row.State != TcpListen {
			t.Logf("%d: %d: [%v]:%d => [%v]:%d %s", i, row.PID, row.LocalAddress, row.LocalPort, row.RemoteAddress, row.RemotePort, row.State)
		} else {
			t.Logf("%d: %d: [%v]:%d %s", i, row.PID, row.LocalAddress, row.LocalPort, row.State)
		}
	}
}

func TestTCPTableOwnerModule(t *testing.T) {
	table, err := GetTCPTableIP4OwnerModule(true, TCPTableAll)
	if err != nil {
		t.Fatal(err)
	}
	for i, row := range table {
		if row.State != TcpListen {
			t.Logf("%d: %d (%s,%s): %v:%d => %v:%d %s", i, row.PID, row.ModuleName, row.ModulePath, row.LocalAddress, row.LocalPort, row.RemoteAddress, row.RemotePort, row.State)
		} else {
			t.Logf("%d: %d (%s,%s): %v:%d %s", i, row.PID, row.ModuleName, row.ModulePath, row.LocalAddress, row.LocalPort, row.State)
		}
	}
}

func TestTCP6TableOwnerModule(t *testing.T) {
	table, err := GetTCPTableIP6OwnerModule(true, TCPTableAll)
	if err != nil {
		t.Fatal(err)
	}
	for i, row := range table {
		if row.State != TcpListen {
			t.Logf("%d: %d (%s,%s): [%v]:%d => [%v]:%d %s", i, row.PID, row.ModuleName, row.ModulePath, row.LocalAddress, row.LocalPort, row.RemoteAddress, row.RemotePort, row.State)
		} else {
			t.Logf("%d: %d (%s,%s): [%v]:%d %s", i, row.PID, row.ModuleName, row.ModulePath, row.LocalAddress, row.LocalPort, row.State)
		}
	}
}
