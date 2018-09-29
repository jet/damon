// +build windows

package main

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

func eatCPU(exitCh <-chan struct{}, doneCh chan struct{}) {
	defer close(doneCh)
	for cpu := 0; cpu < runtime.NumCPU(); cpu++ {
		go func(cpu int) {
			t0 := time.Now()
			for {
				select {
				case <-exitCh:
					return
				default:
					// hot loop
					t1 := time.Now()
					if t1.Sub(t0) > (1 * time.Second) {
						// tick every second
						fmt.Fprintf(os.Stderr, ".")
						t0 = t1
					}
				}
			}
		}(cpu)
	}
}
