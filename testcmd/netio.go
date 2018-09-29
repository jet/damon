// +build windows

package main

import (
	"log"
	"math/rand"
	"net"
	"time"
)

const NetBufferSize = 1024

func eatNetIO(exitCh <-chan struct{}, doneCh chan struct{}) {
	defer close(doneCh)
	listener, err := net.Listen("tcp", ":")
	defer listener.Close()
	if err != nil {
		return
	}
	// Reader
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		var r int
		t0 := time.Now()
		for {
			select {
			case <-doneCh:
				return
			default:
				t1 := time.Now()
				if d := t1.Sub(t0); d > (1 * time.Second) {
					log.Printf("Read: %.2f/Kbps", float64(r)/float64(d.Seconds())/1024.0)
					t0 = t1
					r = 0
				}
				buffer := make([]byte, NetBufferSize)
				n, err := conn.Read(buffer)
				LogErrorf(err, "read failed")
				r += n
			}
		}
	}()
	// Writer
	addr := listener.Addr()
	conn, err := net.Dial(addr.Network(), addr.String())
	defer conn.Close()
	if err != nil {
		return
	}
	t0 := time.Now()
	var w int
	for {
		select {
		case <-exitCh:
			return
		default:
			t1 := time.Now()
			if d := t1.Sub(t0); d > (1 * time.Second) {
				log.Printf("Write: %.2f/Kbps", float64(w)/float64(d.Seconds())/1024.0)
				t0 = t1
				w = 0
			}
			block := make([]byte, NetBufferSize)
			_, err := rand.Read(block)
			LogErrorf(err, "read failed")
			n, err := conn.Write(block)
			LogErrorf(err, "write failed")
			w += n
		}
	}
}
