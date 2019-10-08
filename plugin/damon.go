package plugin

import (
	// "fmt"

	"fmt"
	"io"
	"os/exec"
	"strings"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/lib/fifo"
	"github.com/jet/damon/container"

	"github.com/hashicorp/nomad/plugins/drivers"
)

type damonExec struct {
	cmd        *exec.Cmd
	ccfg       *container.Config
	taskConfig TaskConfig
	cfg        *drivers.TaskConfig
	stdout     io.WriteCloser
	stderr     io.WriteCloser
	logger     log.Logger
}

type hcLogWrapper struct {
	Logger log.Logger
}

func (l hcLogWrapper) Logln(v ...interface{}) {
	if l.Logger == nil {
		return
	}
	l.Logger.Debug(fmt.Sprint(v...))
}
func (l hcLogWrapper) Error(err error, msg string) {
	if l.Logger == nil {
		return
	}
	l.Logger.Error(msg, "error", err)
}

func getCPUMHz(cfg *drivers.TaskConfig, taskConfig TaskConfig, logger log.Logger) int {
	if taskConfig.CPULimit > 0 {
		return taskConfig.CPULimit
	}
	return int(cfg.Resources.NomadResources.Cpu.CpuShares)
}

func getMemoryMB(cfg *drivers.TaskConfig, taskConfig TaskConfig, logger log.Logger) int {
	if taskConfig.MemoryLimit > 0 {
		return taskConfig.MemoryLimit
	}
	return int(cfg.Resources.NomadResources.Memory.MemoryMB)
}

func newDamonExec(cfg *drivers.TaskConfig, taskConfig TaskConfig, logger log.Logger) (*damonExec, error) {
	var d damonExec
	d.ccfg = &container.Config{
		Name:            cfg.ID,
		EnforceCPU:      taskConfig.EnforceCPULimit,
		CPUMHzLimit:     getCPUMHz(cfg, taskConfig, logger),
		EnforceMemory:   taskConfig.EnforceMemoryLimit,
		MemoryMBLimit:   getMemoryMB(cfg, taskConfig, logger),
		RestrictedToken: taskConfig.RestrictedToken,
		CPUHardCap:      true,
		Logger:          hcLogWrapper{Logger: logger},
	}
	d.cfg = cfg
	d.taskConfig = taskConfig
	d.cmd = exec.Command(taskConfig.Command, taskConfig.Args...)
	d.cmd.Dir = cfg.TaskDir().Dir
	d.cmd.Env = cfg.EnvList()
	d.logger = logger
	return &d, nil
}

func (d *damonExec) startContainer(commandCfg *drivers.TaskConfig) (*taskHandle, error) {
	d.logger.Debug("running executable", "task_id", commandCfg.ID, "command", d.cmd.Path, "args", strings.Join(d.cmd.Args, " "))
	stdout, err := d.Stdout()
	if err != nil {
		return nil, err
	}
	stderr, err := d.Stderr()
	if err != nil {
		return nil, err
	}
	cmd := d.cmd
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	c, err := container.RunContained(d.cmd, d.ccfg)
	if err != nil {
		defer d.Close()
		return nil, err
	}
	return &taskHandle{
		container:  c,
		pid:        c.PID,
		logger:     d.logger,
		taskConfig: commandCfg,
		startedAt:  c.StartTime,
		procState:  drivers.TaskStateRunning,
	}, nil
}

func (d *damonExec) Stdout() (io.Writer, error) {
	if d.stdout == nil {
		if d.cfg.StdoutPath == "" {
			return DevNull, nil
		}
		stdout, err := fifo.OpenWriter(d.cfg.StdoutPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open stdout fifo '%s': %v", d.cfg.StdoutPath, err)
		}
		d.logger.Trace("stdout fifo opened", "path", "task_id", d.cfg.ID, d.cfg.StderrPath)
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
			return nil, fmt.Errorf("failed to open stderr fifo '%s': %v", d.cfg.StderrPath, err)
		}
		d.logger.Trace("stderr fifo opened", "path", "task_id", d.cfg.ID, d.cfg.StderrPath)
		d.stderr = stderr
	}
	return d.stderr, nil
}

func (d *damonExec) Close() {
	d.logger.Trace("damon closed", "task_id", d.cfg.ID)
	if d.stdout != nil {
		d.logger.Trace("stdout fifo closed", "task_id", d.cfg.ID)
		d.stdout.Close()
	}
	if d.stderr != nil {
		d.logger.Trace("stderr fifo closed", "task_id", d.cfg.ID)
		d.stderr.Close()
	}
}
