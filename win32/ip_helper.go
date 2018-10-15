// +build windows

package win32

import (
	"net"
	"unsafe"

	"github.com/pkg/errors"
)

type TCPTableInclude int

const (
	// do not reorder
	TCPTableAll TCPTableInclude = iota
	TCPTableConnection
	TCPTableListener
)

type TCPTableIPVersion int

const (
	// do not reorder
	TCPTableIP4 TCPTableIPVersion = iota
	TCPTableIP6
)

type TCPState int

func (s TCPState) String() string {
	return tcpStateToString[s]
}

const (
	//do not reorder
	TcpClosed TCPState = iota + 1
	TcpListen
	TcpSynSent
	TcpSynReceived
	TcpEstablisthed
	TcpFinWait1
	TcpFinWait2
	TcpCloseWait
	TcpClosing
	TcpLastAck
	TcpTimeWait
	TcpDeleteTCB
)

var tcpStateToString = map[TCPState]string{
	TcpClosed:       "CLOSED",
	TcpListen:       "LISTEN",
	TcpSynSent:      "SYN_SENT",
	TcpSynReceived:  "SYN_RECVD",
	TcpEstablisthed: "ESTABLISHED",
	TcpFinWait1:     "FIN_WAIT_1",
	TcpFinWait2:     "FIN_WAIT_2",
	TcpCloseWait:    "CLOSE_WAIT",
	TcpClosing:      "CLOSING",
	TcpLastAck:      "LAST_ACK",
	TcpTimeWait:     "TIME_WAIT",
	TcpDeleteTCB:    "DELETE_TCB",
}

type TCPOwnerModuleConnection struct {
	TCPConnection
	PID        int
	ModulePath string
	ModuleName string
}
type TCPOwnerConnection struct {
	TCPConnection
	PID int
}
type TCPConnection struct {
	RemoteAddress net.IP
	RemotePort    uint16
	RemoteScopeID uint32
	LocalAddress  net.IP
	LocalPort     uint16
	LocalScopeID  uint32
	State         TCPState
}

func GetTCPTableIP4OwnerPID(order bool, inc TCPTableInclude) ([]TCPOwnerConnection, error) {
	var tblClass = _TCP_TABLE_OWNER_PID_ALL
	switch inc {
	case TCPTableListener:
		tblClass = _TCP_TABLE_OWNER_PID_LISTENER
	case TCPTableConnection:
		tblClass = _TCP_TABLE_OWNER_PID_CONNECTIONS
	case TCPTableAll:
		tblClass = _TCP_TABLE_OWNER_PID_ALL
	}
	buf, err := getExtendedTcpTable(order, _AF_INET, tblClass)
	if err != nil {
		return nil, err
	}
	var table []TCPOwnerConnection
	pTable := (*_MIB_TCPTABLE_OWNER_PID)(unsafe.Pointer(&buf[0]))
	for i := uint32(0); i < pTable.dwNumEntries; i++ {
		pRow := (*_MIB_TCPROW_OWNER_PID)(unsafe.Pointer(uintptr(unsafe.Pointer(&pTable.table[0])) + uintptr(i)*unsafe.Sizeof(pTable.table[0])))
		row := TCPOwnerConnection{
			TCPConnection: TCPConnection{
				RemoteAddress: net.IP(pRow.dwRemoteAddr[:]),
				RemotePort:    dwToPort(pRow.dwRemotePort),
				LocalAddress:  net.IP(pRow.dwLocalAddr[:]),
				LocalPort:     dwToPort(pRow.dwLocalPort),
				State:         TCPState(pRow.dwState),
			},
			PID: int(pRow.dwOwningPid),
		}
		table = append(table, row)
	}
	return table, nil
}

func GetTCPTableIP6OwnerPID(order bool, inc TCPTableInclude) ([]TCPOwnerConnection, error) {
	var tblClass = _TCP_TABLE_OWNER_PID_ALL
	switch inc {
	case TCPTableListener:
		tblClass = _TCP_TABLE_OWNER_PID_LISTENER
	case TCPTableConnection:
		tblClass = _TCP_TABLE_OWNER_PID_CONNECTIONS
	case TCPTableAll:
		tblClass = _TCP_TABLE_OWNER_PID_ALL
	}
	buf, err := getExtendedTcpTable(order, _AF_INET6, tblClass)
	if err != nil {
		return nil, err
	}
	var table []TCPOwnerConnection
	pTable := (*_MIB_TCP6TABLE_OWNER_PID)(unsafe.Pointer(&buf[0]))
	for i := uint32(0); i < pTable.dwNumEntries; i++ {
		pRow := (*_MIB_TCP6ROW_OWNER_PID)(unsafe.Pointer(uintptr(unsafe.Pointer(&pTable.table[0])) + uintptr(i)*unsafe.Sizeof(pTable.table[0])))
		row := TCPOwnerConnection{
			TCPConnection: TCPConnection{
				RemoteAddress: net.IP(pRow.ucRemoteAddr[:]),
				RemotePort:    dwToPort(pRow.dwRemotePort),
				RemoteScopeID: pRow.dwRemoteScopeId,
				LocalAddress:  net.IP(pRow.ucLocalAddr[:]),
				LocalPort:     dwToPort(pRow.dwLocalPort),
				LocalScopeID:  pRow.dwLocalScopeId,
				State:         TCPState(pRow.dwState),
			},
			PID: int(pRow.dwOwningPid),
		}
		table = append(table, row)
	}
	return table, nil
}

func GetTCPTableIP4OwnerModule(order bool, inc TCPTableInclude) ([]TCPOwnerModuleConnection, error) {
	var tblClass = _TCP_TABLE_OWNER_MODULE_ALL
	switch inc {
	case TCPTableListener:
		tblClass = _TCP_TABLE_OWNER_MODULE_LISTENER
	case TCPTableConnection:
		tblClass = _TCP_TABLE_OWNER_MODULE_CONNECTIONS
	case TCPTableAll:
		tblClass = _TCP_TABLE_OWNER_MODULE_ALL
	}
	buf, err := getExtendedTcpTable(order, _AF_INET, tblClass)
	if err != nil {
		return nil, errors.Wrapf(err, "win32: GetTCPTableIP4OwnerModule: getExtendedTcpTable failed")
	}
	var table []TCPOwnerModuleConnection
	pTable := (*_MIB_TCPTABLE_OWNER_MODULE)(unsafe.Pointer(&buf[0]))
	for i := uint32(0); i < pTable.dwNumEntries; i++ {
		pRow := (*_MIB_TCPROW_OWNER_MODULE)(unsafe.Pointer(uintptr(unsafe.Pointer(&pTable.table[0])) + uintptr(i)*unsafe.Sizeof(pTable.table[0])))
		row := TCPOwnerModuleConnection{
			TCPConnection: TCPConnection{
				RemoteAddress: net.IP(pRow.dwRemoteAddr[:]),
				RemotePort:    dwToPort(pRow.dwRemotePort),
				LocalAddress:  net.IP(pRow.dwLocalAddr[:]),
				LocalPort:     dwToPort(pRow.dwLocalPort),
				State:         TCPState(pRow.dwState),
			},
			PID: int(pRow.dwOwningPid),
		}
		info, err := getOwnerModuleFromTcpEntry(pRow)
		LogError(err, "win32: GetTCPTableIP4OwnerModule: getOwnerModuleFromTcpEntry failed")
		if info != nil {
			row.ModuleName = UTF16PtrToString(info.pModuleName)
			row.ModulePath = UTF16PtrToString(info.pModulePath)
		}
		table = append(table, row)
	}
	return table, nil
}

func GetTCPTableIP6OwnerModule(order bool, inc TCPTableInclude) ([]TCPOwnerModuleConnection, error) {
	var tblClass = _TCP_TABLE_OWNER_MODULE_ALL
	switch inc {
	case TCPTableListener:
		tblClass = _TCP_TABLE_OWNER_MODULE_LISTENER
	case TCPTableConnection:
		tblClass = _TCP_TABLE_OWNER_MODULE_CONNECTIONS
	case TCPTableAll:
		tblClass = _TCP_TABLE_OWNER_MODULE_ALL
	}
	buf, err := getExtendedTcpTable(order, _AF_INET6, tblClass)
	if err != nil {
		return nil, errors.Wrapf(err, "win32: GetTCPTableIP6OwnerModule: getExtendedTcpTable failed")
	}
	var table []TCPOwnerModuleConnection
	pTable := (*_MIB_TCP6TABLE_OWNER_MODULE)(unsafe.Pointer(&buf[0]))
	for i := uint32(0); i < pTable.dwNumEntries; i++ {
		pRow := (*_MIB_TCP6ROW_OWNER_MODULE)(unsafe.Pointer(uintptr(unsafe.Pointer(&pTable.table[0])) + uintptr(i)*unsafe.Sizeof(pTable.table[0])))
		row := TCPOwnerModuleConnection{
			TCPConnection: TCPConnection{
				RemoteAddress: net.IP(pRow.ucRemoteAddr[:]),
				RemotePort:    dwToPort(pRow.dwRemotePort),
				RemoteScopeID: pRow.dwRemoteScopeId,
				LocalAddress:  net.IP(pRow.ucLocalAddr[:]),
				LocalPort:     dwToPort(pRow.dwLocalPort),
				LocalScopeID:  pRow.dwLocalScopeId,
				State:         TCPState(pRow.dwState),
			},
			PID: int(pRow.dwOwningPid),
		}
		info, err := getOwnerModuleFromTcp6Entry(pRow)
		LogError(err, "win32: GetTCPTableIP6OwnerModule: getOwnerModuleFromTcp6Entry failed")
		if info != nil {
			row.ModuleName = UTF16PtrToString(info.pModuleName)
			row.ModulePath = UTF16PtrToString(info.pModulePath)
		}
		table = append(table, row)
	}
	return table, nil
}
