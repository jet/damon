// +build windows

package win32

import (
	"encoding/binary"
	"syscall"
	"unsafe"
)

var (
	procGetExtendedTcpTable         = iphlpapiDLL.NewProc("GetExtendedTcpTable")
	procGetOwnerModuleFromTcpEntry  = iphlpapiDLL.NewProc("GetOwnerModuleFromTcpEntry")
	procGetOwnerModuleFromTcp6Entry = iphlpapiDLL.NewProc("GetOwnerModuleFromTcp6Entry")
)

const (
	_AF_UNSPEC    uint32 = 0
	_AF_INET      uint32 = 2
	_AF_IPX       uint32 = 6
	_AF_APPLETALK uint32 = 16
	_AF_NETBIOS   uint32 = 17
	_AF_INET6     uint32 = 23
	_AF_IRDA      uint32 = 26
	_AF_BTH       uint32 = 32
)

// typedef struct _MIB_TCPTABLE_OWNER_PID
// {
//     DWORD                dwNumEntries;
//     MIB_TCPROW_OWNER_PID table[ANY_SIZE];
// } MIB_TCPTABLE_OWNER_PID, *PMIB_TCPTABLE_OWNER_PID;
type _MIB_TCPTABLE_OWNER_PID struct {
	dwNumEntries uint32
	table        [1]_MIB_TCPROW_OWNER_PID
}

// typedef struct _MIB_TCPROW_OWNER_PID
// {
//     DWORD       dwState;
//     DWORD       dwLocalAddr;
//     DWORD       dwLocalPort;
//     DWORD       dwRemoteAddr;
//     DWORD       dwRemotePort;
//     DWORD       dwOwningPid;
// } MIB_TCPROW_OWNER_PID, *PMIB_TCPROW_OWNER_PID;
// https://docs.microsoft.com/en-us/windows/desktop/api/tcpmib/ns-tcpmib-_mib_tcprow_owner_pid
type _MIB_TCPROW_OWNER_PID struct {
	dwState      uint32
	dwLocalAddr  [4]byte // [4] bytes makes it easier to create an net.IP
	dwLocalPort  uint32
	dwRemoteAddr [4]byte // same.
	dwRemotePort uint32
	dwOwningPid  uint32
}

// typedef struct _MIB_TCPTABLE_OWNER_MODULE {
//   DWORD                   dwNumEntries;
//   MIB_TCPROW_OWNER_MODULE table[ANY_SIZE];
// } MIB_TCPTABLE_OWNER_MODULE, *PMIB_TCPTABLE_OWNER_MODULE;
// https://docs.microsoft.com/en-us/windows/desktop/api/tcpmib/ns-tcpmib-_mib_tcptable_owner_module
type _MIB_TCPTABLE_OWNER_MODULE struct {
	dwNumEntries uint32
	table        [1]_MIB_TCPROW_OWNER_MODULE
}

// typedef struct _MIB_TCP6ROW_OWNER_PID {
// 	UCHAR ucLocalAddr[16];
// 	DWORD dwLocalScopeId;
// 	DWORD dwLocalPort;
// 	UCHAR ucRemoteAddr[16];
// 	DWORD dwRemoteScopeId;
// 	DWORD dwRemotePort;
// 	DWORD dwState;
// 	DWORD dwOwningPid;
// } MIB_TCP6ROW_OWNER_PID, *PMIB_TCP6ROW_OWNER_PID;
// https://docs.microsoft.com/en-us/windows/desktop/api/tcpmib/ns-tcpmib-_mib_tcp6row_owner_pid
type _MIB_TCP6ROW_OWNER_PID struct {
	ucLocalAddr     [16]byte
	dwLocalScopeId  uint32
	dwLocalPort     uint32
	ucRemoteAddr    [16]byte
	dwRemoteScopeId uint32
	dwRemotePort    uint32
	dwState         uint32
	dwOwningPid     uint32
}

// typedef struct _MIB_TCP6TABLE_OWNER_PID {
// 	DWORD                 dwNumEntries;
// 	MIB_TCP6ROW_OWNER_PID table[ANY_SIZE];
// } MIB_TCP6TABLE_OWNER_PID, *PMIB_TCP6TABLE_OWNER_PID;
// https://docs.microsoft.com/en-us/windows/desktop/api/tcpmib/ns-tcpmib-_mib_tcp6table_owner_pid
type _MIB_TCP6TABLE_OWNER_PID struct {
	dwNumEntries uint32
	table        [1]_MIB_TCP6ROW_OWNER_PID
}

// typedef struct _MIB_TCP6ROW_OWNER_MODULE {
// 	UCHAR         ucLocalAddr[16];
// 	DWORD         dwLocalScopeId;
// 	DWORD         dwLocalPort;
// 	UCHAR         ucRemoteAddr[16];
// 	DWORD         dwRemoteScopeId;
// 	DWORD         dwRemotePort;
// 	DWORD         dwState;
// 	DWORD         dwOwningPid;
// 	LARGE_INTEGER liCreateTimestamp;
// 	ULONGLONG     OwningModuleInfo[TCPIP_OWNING_MODULE_SIZE];
// } MIB_TCP6ROW_OWNER_MODULE, *PMIB_TCP6ROW_OWNER_MODULE;
// https://docs.microsoft.com/en-us/windows/desktop/api/tcpmib/ns-tcpmib-_mib_tcp6row_owner_module
type _MIB_TCP6ROW_OWNER_MODULE struct {
	ucLocalAddr       [16]byte
	dwLocalScopeId    uint32
	dwLocalPort       uint32
	ucRemoteAddr      [16]byte
	dwRemoteScopeId   uint32
	dwRemotePort      uint32
	dwState           uint32
	dwOwningPid       uint32
	liCreateTimestamp uint64
	OwningModuleInfo  [TCPIP_OWNING_MODULE_SIZE]uint64
}

// typedef struct _MIB_TCP6TABLE_OWNER_MODULE {
// 	DWORD                    dwNumEntries;
// 	MIB_TCP6ROW_OWNER_MODULE table[ANY_SIZE];
// } MIB_TCP6TABLE_OWNER_MODULE, *PMIB_TCP6TABLE_OWNER_MODULE;
// https://docs.microsoft.com/en-us/windows/desktop/api/tcpmib/ns-tcpmib-_mib_tcp6table_owner_module
type _MIB_TCP6TABLE_OWNER_MODULE struct {
	dwNumEntries uint32
	table        [1]_MIB_TCP6ROW_OWNER_MODULE
}

// typedef struct _MIB_TCPROW_OWNER_MODULE {
// 	DWORD         dwState;
// 	DWORD         dwLocalAddr;
// 	DWORD         dwLocalPort;
// 	DWORD         dwRemoteAddr;
// 	DWORD         dwRemotePort;
// 	DWORD         dwOwningPid;
// 	LARGE_INTEGER liCreateTimestamp;
// 	ULONGLONG     OwningModuleInfo[TCPIP_OWNING_MODULE_SIZE];
// } MIB_TCPROW_OWNER_MODULE, *PMIB_TCPROW_OWNER_MODULE;
// https://docs.microsoft.com/en-us/windows/desktop/api/tcpmib/ns-tcpmib-_mib_tcprow_owner_module
type _MIB_TCPROW_OWNER_MODULE struct {
	dwState           uint32
	dwLocalAddr       [4]byte // [4] bytes makes it easier to create an net.IP
	dwLocalPort       uint32
	dwRemoteAddr      [4]byte // same.
	dwRemotePort      uint32
	dwOwningPid       uint32
	liCreateTimestamp int64
	OwningModuleInfo  [TCPIP_OWNING_MODULE_SIZE]uint64
}

const TCPIP_OWNING_MODULE_SIZE = 16

// typedef struct _TCPIP_OWNER_MODULE_BASIC_INFO {
//   PWCHAR pModuleName;
//   PWCHAR pModulePath;
// } TCPIP_OWNER_MODULE_BASIC_INFO, *PTCPIP_OWNER_MODULE_BASIC_INFO;
// https://docs.microsoft.com/en-us/windows/desktop/api/iprtrmib/ns-iprtrmib-_tcpip_owner_module_basic_info
type _TCPIP_OWNER_MODULE_BASIC_INFO struct {
	pModuleName *uint16
	pModulePath *uint16
}

const (
	// do not reorder
	_TCP_TABLE_BASIC_LISTENER uint32 = iota
	_TCP_TABLE_BASIC_CONNECTIONS
	_TCP_TABLE_BASIC_ALL
	_TCP_TABLE_OWNER_PID_LISTENER
	_TCP_TABLE_OWNER_PID_CONNECTIONS
	_TCP_TABLE_OWNER_PID_ALL
	_TCP_TABLE_OWNER_MODULE_LISTENER
	_TCP_TABLE_OWNER_MODULE_CONNECTIONS
	_TCP_TABLE_OWNER_MODULE_ALL
)

// typedef enum {
//     MIB_TCP_STATE_CLOSED     =  1,
//     MIB_TCP_STATE_LISTEN     =  2,
//     MIB_TCP_STATE_SYN_SENT   =  3,
//     MIB_TCP_STATE_SYN_RCVD   =  4,
//     MIB_TCP_STATE_ESTAB      =  5,
//     MIB_TCP_STATE_FIN_WAIT1  =  6,
//     MIB_TCP_STATE_FIN_WAIT2  =  7,
//     MIB_TCP_STATE_CLOSE_WAIT =  8,
//     MIB_TCP_STATE_CLOSING    =  9,
//     MIB_TCP_STATE_LAST_ACK   = 10,
//     MIB_TCP_STATE_TIME_WAIT  = 11,
//     MIB_TCP_STATE_DELETE_TCB = 12,
//     //
//     // Extra TCP states not defined in the MIB
//     //
//     MIB_TCP_STATE_RESERVED      = 100
// } MIB_TCP_STATE;
type _MIB_TCP_STATE uint32

const (
	_MIB_TCP_STATE_CLOSED _MIB_TCP_STATE = iota + 1
	_MIB_TCP_STATE_LISTEN
	_MIB_TCP_STATE_SYN_SENT
	_MIB_TCP_STATE_SYN_RCVD
	_MIB_TCP_STATE_ESTAB
	_MIB_TCP_STATE_FIN_WAIT1
	_MIB_TCP_STATE_FIN_WAIT2
	_MIB_TCP_STATE_CLOSE_WAIT
	_MIB_TCP_STATE_CLOSING
	_MIB_TCP_STATE_LAST_ACK
	_MIB_TCP_STATE_TIME_WAIT
	_MIB_TCP_STATE_DELETE_TCB

	_MIB_TCP_STATE_RESERVED _MIB_TCP_STATE = 100
)

// DWORD GetExtendedTcpTable(
// 	PVOID           pTcpTable,
// 	PDWORD          pdwSize,
// 	BOOL            bOrder,
// 	ULONG           ulAf,
// 	TCP_TABLE_CLASS TableClass,
// 	ULONG           Reserved
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/iphlpapi/nf-iphlpapi-getextendedtcptable
func getExtendedTcpTable(order bool, ulAf uint32, tableClass uint32) ([]byte, error) {
	var buffer []byte
	var pTcpTable *byte
	var dwSize uint32
	for {
		ret, _, errno := procGetExtendedTcpTable.Call(
			uintptr(unsafe.Pointer(pTcpTable)),
			uintptr(unsafe.Pointer(&dwSize)),
			uintptr(toBOOL(order)),
			uintptr(ulAf),
			uintptr(tableClass),
			uintptr(uint32(0)),
		)
		if ret != NO_ERROR {
			if syscall.Errno(ret) == syscall.ERROR_INSUFFICIENT_BUFFER {
				buffer = make([]byte, int(dwSize))
				pTcpTable = &buffer[0]
				continue
			}
			return nil, errnoToError(errno)
		}
		return buffer, nil
	}
}

const (
	TCPIP_OWNER_MODULE_INFO_BASIC = 0
)

// DWORD GetOwnerModuleFromTcpEntry(
//   PMIB_TCPROW_OWNER_MODULE      pTcpEntry,
//   TCPIP_OWNER_MODULE_INFO_CLASS Class,
//   PVOID                         pBuffer,
//   PDWORD                        pdwSize
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/iphlpapi/nf-iphlpapi-getownermodulefromtcpentry
func getOwnerModuleFromTcpEntry(pTcpEntry *_MIB_TCPROW_OWNER_MODULE) (*_TCPIP_OWNER_MODULE_BASIC_INFO, error) {
	var buffer []byte
	var pBuffer *byte
	var dwSize uint32
	for {
		ret, _, errno := procGetOwnerModuleFromTcpEntry.Call(
			uintptr(unsafe.Pointer(pTcpEntry)),
			uintptr(TCPIP_OWNER_MODULE_INFO_BASIC),
			uintptr(unsafe.Pointer(pBuffer)),
			uintptr(unsafe.Pointer(&dwSize)),
		)
		if ret != NO_ERROR {
			if syscall.Errno(ret) == syscall.ERROR_INSUFFICIENT_BUFFER {
				buffer = make([]byte, int(dwSize))
				pBuffer = &buffer[0]
				continue
			}
			return nil, errnoToError(errno)
		}
		return (*_TCPIP_OWNER_MODULE_BASIC_INFO)(unsafe.Pointer(pBuffer)), nil
	}
}

// DWORD GetOwnerModuleFromTcp6Entry(
// 	PMIB_TCP6ROW_OWNER_MODULE     pTcpEntry,
// 	TCPIP_OWNER_MODULE_INFO_CLASS Class,
// 	PVOID                         pBuffer,
// 	PDWORD                        pdwSize
// );
// https://docs.microsoft.com/en-us/windows/desktop/api/iphlpapi/nf-iphlpapi-getownermodulefromtcp6entry
func getOwnerModuleFromTcp6Entry(pTcpEntry *_MIB_TCP6ROW_OWNER_MODULE) (*_TCPIP_OWNER_MODULE_BASIC_INFO, error) {
	var buffer []byte
	var pBuffer *byte
	var dwSize uint32
	for {
		ret, _, errno := procGetOwnerModuleFromTcpEntry.Call(
			uintptr(unsafe.Pointer(pTcpEntry)),
			uintptr(TCPIP_OWNER_MODULE_INFO_BASIC),
			uintptr(unsafe.Pointer(pBuffer)),
			uintptr(unsafe.Pointer(&dwSize)),
		)
		if ret != NO_ERROR {
			if syscall.Errno(ret) == syscall.ERROR_INSUFFICIENT_BUFFER {
				buffer = make([]byte, int(dwSize))
				pBuffer = &buffer[0]
				continue
			}
			return nil, errnoToError(errno)
		}
		return (*_TCPIP_OWNER_MODULE_BASIC_INFO)(unsafe.Pointer(pBuffer)), nil
	}
}

func dwToPort(dw uint32) uint16 {
	if dw > 0 { // Transform from Network ByteOrder
		bs := make([]byte, 2)
		binary.LittleEndian.PutUint16(bs, uint16(dw))
		return uint16(binary.BigEndian.Uint16(bs))
	}
	return 0
}
