package container

import (
	"context"
	"io"

	"github.com/jet/damon/win32"
)

type TaskConfig struct {
	Command []string
	Dir     string
	EnvList []string
	Stdout  io.Writer
	Stderr  io.Writer
}

type Task struct {
	osProcess *win32.Process
}

func (t *Task) Wait(ctx context.Context) (int, error) {
	exitCh := make(chan int, 1)
	errCh := make(chan error, 1)
	go func() {
		defer close(exitCh)
		defer close(errCh)
		res, err := t.osProcess.Wait()
		if err != nil {
			errCh <- err
			return
		}
		if res.Err != nil {
			errCh <- res.Err
			return
		}
		exitCh <- res.ExitStatus
	}()
	select {
	case err := <-errCh:
		t.osProcess.Kill()
		return -1, err
	case res := <-exitCh:
		return res, nil
	case <-ctx.Done():
		t.osProcess.Kill()
		return -1, ctx.Err()
	}
}
