// +build windows

package main

import (
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"
)

const DiskBufferSize = 1024

func eatDiskIO(exitCh <-chan struct{}, doneCh chan struct{}) {
	defer close(doneCh)
	fd, err := ioutil.TempFile("", "eatDiskIO-*.tmp")
	if err != nil {
		LogErrorf(err, "create temp file failed")
		return
	}
	defer func() {
		LogErrorf(fd.Close(), "closed failed")
		LogErrorf(os.Remove(fd.Name()), "remove file failed '%s'", fd.Name())
	}()
	t0 := time.Now()
	var r, w int
	for {
		select {
		case <-exitCh:
			return
		default:
			t1 := time.Now()
			if d := t1.Sub(t0); d > (1 * time.Second) {
				log.Printf("Read/Write: %.2f/%.2f Kbps", float64(r)/float64(d.Seconds())/1024.0, float64(w)/float64(d.Seconds())/1024.0)
				t0 = t1
				w, r = 0, 0
			}
			block := make([]byte, DiskBufferSize)
			rand.Read(block)

			n, err := fd.Write(block)
			w += n
			LogErrorf(err, "write failed")
			_, err = fd.Seek(0, 0)
			LogErrorf(err, "seek failed")

			n, err = fd.Read(block)
			r += n
			LogErrorf(err, "read failed")
			_, err = fd.Seek(0, 0)
			LogErrorf(err, "seek failed")
		}
	}
}
