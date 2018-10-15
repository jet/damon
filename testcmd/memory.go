// +build windows

package main

import (
	"math/rand"
	"time"
)

func eatMemory(exitCh <-chan struct{}, doneCh chan struct{}) {
	defer close(doneCh)
	memory := make([]byte, 0)
	for {
		select {
		case <-exitCh:
			return
		case <-time.After(100 * time.Millisecond):
			block := make([]byte, 512*1024) // 512k
			rand.Read(block)
			memory = append(memory, block...)
		}
	}

}
