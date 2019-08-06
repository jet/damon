package plugin

import (
	// "fmt"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/lib/fifo"
	"github.com/jet/damon/container"

	"github.com/hashicorp/nomad/plugins/drivers"
	// "github.com/jet/damon/log"
)

type damonExec struct {
	argv         []string
	cmd          *exec.Cmd
	ccfg         *container.Config
	cachedir     string
	taskConfig   TaskConfig
	cfg          *drivers.TaskConfig
	stdout       io.WriteCloser
	stderr       io.WriteCloser
	env          []string
	TaskDir      string
	state        *os.ProcessState
	containerPid int
	exitCode     int
	ExitError    error
	logger       hclog.Logger
}

func newDamonExec(cfg *drivers.TaskConfig, taskConfig TaskConfig) (*damonExec, error) {
	var d damonExec
	d.ccfg = &container.Config{
		Name:            cfg.ID,
		EnforceCPU:      taskConfig.EnforceCPULimit,
		EnforceMemory:   taskConfig.EnforceMemoryLimit,
		RestrictedToken: taskConfig.RestrictedToken,
		CPUHardCap:      true,
	}
	d.cfg = cfg
	d.taskConfig = taskConfig
	d.env = cfg.EnvList()
	d.cmd = exec.Command(taskConfig.Command, taskConfig.Args...)
	d.cmd.Dir = cfg.TaskDir().Dir

	return &d, nil
}

func (d *damonExec) startContainer(commandCfg *drivers.TaskConfig) error {
	d.logger.Debug("running executable", d.cmd.Path, strings.Join(d.cmd.Args, " "))
	stdout, err := d.Stdout()
	if err != nil {
		return err
	}
	stderr, err := d.Stderr()
	if err != nil {
		return err
	}
	cmd := d.cmd
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	c, err := container.RunContained(d.cmd, d.ccfg)
	if err != nil {
		return err
	}
	if _, err := c.Wait(nil); err != nil {
		return err
	}
	d.state = cmd.ProcessState
	return nil
}

func (d *damonExec) Stdout() (io.Writer, error) {
	if d.stdout == nil {
		if d.cfg.StdoutPath == "" {
			return DevNull, nil
		}
		stdout, err := fifo.OpenWriter(d.cfg.StdoutPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open stdout fifo: %v", err)
		}
		d.stdout = stdout
	}
	return d.stdout, nil
}

func (d *damonExec) Stderr() (io.Writer, error) {
	if d.stderr == nil {
		if d.cfg.StderrPath == "" {
			return DevNull, nil
		}
		stderr, err := fifo.OpenWriter(d.cfg.StderrPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open stderr fifo: %v", err)
		}
		d.stderr = stderr
	}
	return d.stderr, nil
}

func (d *damonExec) Close() {
	if d.stdout != nil {
		d.stdout.Close()
	}
	if d.stderr != nil {
		d.stderr.Close()
	}
}
