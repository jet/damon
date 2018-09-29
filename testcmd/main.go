// +build windows

package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

func getArgDuration(i int, def time.Duration) time.Duration {
	if len(os.Args) > i {
		d, err := time.ParseDuration(os.Args[i])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing arg %d [%s]: %v", i, os.Args[i], err)
			os.Exit(1)
		}
		return d
	}
	return def
}

func dieOnError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Println("PID:", os.Getpid())
		os.Exit(0)
	}
	rc := realMain()
	fmt.Println("rc", rc)
	os.Exit(rc)
}

func realMain() int {
	cmd := os.Args[1]
	doneCh := make(chan struct{})
	exitCh := make(chan struct{})
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh)
	rc := 0
	switch cmd {
	case "err":
		close(doneCh)
		rc = 1
	case "wait":
		close(doneCh)
		break
	case "wait_nosig":
		time.Sleep(getArgDuration(2, 10*time.Second))
		return 0
	case "batch_login":
		if len(os.Args) > 2 {
			dieOnError(addTestUserRights(os.Args[2], []string{"SeBatchLogonRight"}))
		}
		return 0
	case "cpu":
		go eatCPU(exitCh, doneCh)
	case "mem":
		go eatMemory(exitCh, doneCh)
	case "diskio":
		go eatDiskIO(exitCh, doneCh)
	case "netio":
		go eatNetIO(exitCh, doneCh)
	case "env":
		for _, env := range os.Environ() {
			fmt.Println(env)
		}
	}
	dur := getArgDuration(2, 10*time.Second)
	select {
	case <-sigCh:
		close(exitCh)
		rc = 1
	case <-time.After(dur):
		close(exitCh)
	}
	<-doneCh
	return rc
}
