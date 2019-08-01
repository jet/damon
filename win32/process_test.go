// +build windows

package win32

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func SkipIfDocker(t *testing.T) {
	t.Helper()
	if d := os.Getenv("DOCKER"); d != "" && d != "no" {
		t.Skip("SkipIfDocker")
	}
}

func TestRunProcess(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := exec.Command(SetupTestExe(t))
	token, err := CurrentProcessToken()
	if err != nil {
		t.Fatal("CurrentProcessToken", err)
	}
	cmd.Stdout = buf
	defer token.Close()
	proc, err := StartProcess(cmd, AccessToken(token))
	if err != nil {
		t.Fatal("StartProcess()", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := proc.Wait(ctx.Done())
	if err != nil {
		t.Fatal("proc.Wait()", err)
	}
	if rc := res.ExitStatus; rc != 0 {
		t.Fatalf("res.ExitStatus != 0: %d", rc)
	}
	t.Log("out", buf.String())
}

func TestRunProcessWaitSignal(t *testing.T) {
	SkipIfDocker(t)
	buf := &bytes.Buffer{}
	cmd := exec.Command(SetupTestExe(t), "wait")
	token, err := CurrentProcessToken()
	if err != nil {
		t.Fatal("CurrentProcessToken", err)
	}
	cmd.Stdout = buf
	defer token.Close()
	proc, err := StartProcess(cmd, AccessToken(token))
	if err != nil {
		t.Fatal("StartProcess()", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	res, err := proc.Wait(ctx.Done())
	if err != nil {
		t.Fatal("proc.Wait()", err)
	}
	if rc := res.ExitStatus; rc != 1 {
		t.Fatalf("res.ExitStatus != 1: %d", rc)
	}
	out := strings.TrimSpace(buf.String())
	exp := "rc 1"
	t.Log("out", out)
	if out != exp {
		t.Fatalf("out: expected '%s', actual '%s'", exp, out)
	}
}

func TestRunProcessWaitNoSignal(t *testing.T) {
	SkipIfDocker(t)
	buf := &bytes.Buffer{}
	cmd := exec.Command(SetupTestExe(t), "wait_nosig")
	token, err := CurrentProcessToken()
	if err != nil {
		t.Fatal("CurrentProcessToken", err)
	}
	cmd.Stdout = buf
	defer token.Close()
	proc, err := StartProcess(cmd, AccessToken(token))
	if err != nil {
		t.Fatal("StartProcess()", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	res, err := proc.Wait(ctx.Done())
	if err != nil {
		t.Fatal("proc.Wait()", err)
	}
	if rc := res.ExitStatus; rc != 0 {
		t.Fatalf("res.ExitStatus != 0: %d", rc)
	}
	out := strings.TrimSpace(buf.String())
	exp := "rc 0"
	t.Log("out", out)
	if out != exp {
		t.Fatalf("out: expected '%s', actual '%s'", exp, out)
	}
}
