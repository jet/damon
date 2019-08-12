package plugin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	log "github.com/hashicorp/go-hclog"
	cstructs "github.com/hashicorp/nomad/client/structs"
	"github.com/hashicorp/nomad/drivers/shared/eventer"
	"github.com/hashicorp/nomad/plugins/base"
	"github.com/hashicorp/nomad/plugins/drivers"
	"github.com/hashicorp/nomad/plugins/shared/hclspec"
	pstructs "github.com/hashicorp/nomad/plugins/shared/structs"
	"github.com/jet/damon/container"
	"github.com/jet/damon/version"
)

func NewDriverPlugin(logger log.Logger) *DriverPlugin {
	c, err := wrapJob(logger)
	if err != nil {
		logger.Error("error assigning job object", "error", err)
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &DriverPlugin{
		logger: logger,
		ctx:    ctx,
		signalShutdown: func() {
			logger.Info("shutting down damon plugin")
			c.Close()
			cancel()
		},
		eventer:   eventer.NewEventer(ctx, logger),
		taskStore: newTaskStore(),
	}
}

const (
	// pluginName is the name of the plugin
	pluginName = "damon"
	// fingerprintPeriod is the interval at which the driver will send fingerprint responses
	fingerprintPeriod = 30 * time.Second
	// taskHandleVersion is the version of task handle which this driver sets
	// and understands how to decode driver state
	taskHandleVersion = 1
)

var (
	pluginInfo = &base.PluginInfoResponse{
		Name:          pluginName,
		Type:          base.PluginTypeDriver,
		PluginVersion: version.Number,
		PluginApiVersions: []string{
			drivers.ApiVersion010,
		},
	}
	configSpec = hclspec.NewObject(map[string]*hclspec.Spec{
		"enabled": hclspec.NewDefault(
			hclspec.NewAttr("enabled", "bool", false),
			hclspec.NewLiteral("true"),
		),
	})
	taskConfigSpec = hclspec.NewObject(map[string]*hclspec.Spec{
		"command": hclspec.NewAttr("command", "string", true),
		"args":    hclspec.NewAttr("args", "list(string)", false),
		"enforce_cpu_limit": hclspec.NewDefault(
			hclspec.NewAttr("cpu_limit", "bool", false),
			hclspec.NewLiteral("true"),
		),
		"enforce_memory_limit": hclspec.NewDefault(
			hclspec.NewAttr("enforce_memory_limit", "bool", false),
			hclspec.NewLiteral("true"),
		),
		"restricted_token": hclspec.NewDefault(
			hclspec.NewAttr("restricted_token", "bool", false),
			hclspec.NewLiteral("true"),
		),
	})
	capabilities = &drivers.Capabilities{
		SendSignals: true,
		Exec:        true,
		FSIsolation: drivers.FSIsolationNone,
	}
)

type DriverPlugin struct {
	ctx            context.Context
	signalShutdown context.CancelFunc
	eventer        *eventer.Eventer
	config         *Config
	nomadConfig    *base.ClientDriverConfig
	taskStore      *taskStore
	logger         log.Logger
}

type Config struct {
	Enabled            bool `codec:"enabled"`
	EnforceCPULimit    bool `codec:"enforce_cpu_limit"`
	EnforceMemoryLimit bool `codec:"enforce_memory_limit"`
	RestrictedToken    bool `codec:"restricted_token"`
}

type TaskConfig struct {
	Command            string   `codec:"command"`
	Args               []string `codec:"args"`
	EnforceCPULimit    bool     `codec:"enforce_cpu_limit"`
	EnforceMemoryLimit bool     `codec:"enforce_memory_limit"`
	RestrictedToken    bool     `codec:"restricted_token"`
	CPULimit           int      `codec:"cpu_limit"`
	MemoryLimit        int      `codec:"memory_limit"`
}

// PluginInfo describes the type and version of a plugin.
func (d *DriverPlugin) PluginInfo() (*base.PluginInfoResponse, error) {
	return pluginInfo, nil
}

// ConfigSchema returns the schema for parsing the plugins configuration.
func (d *DriverPlugin) ConfigSchema() (*hclspec.Spec, error) {
	return configSpec, nil
}

// SetConfig is used to set the configuration by passing a MessagePack
// encoding of it.
func (d *DriverPlugin) SetConfig(cfg *base.Config) error {
	var config Config
	if len(cfg.PluginConfig) != 0 {
		if err := base.MsgPackDecode(cfg.PluginConfig, &config); err != nil {
			return err
		}
	}
	d.config = &config
	if cfg.AgentConfig != nil {
		d.nomadConfig = cfg.AgentConfig.Driver
	}
	return nil
}

func (d *DriverPlugin) TaskConfigSchema() (*hclspec.Spec, error) {
	return taskConfigSpec, nil
}

func (d *DriverPlugin) Capabilities() (*drivers.Capabilities, error) {
	return capabilities, nil
}

func (d *DriverPlugin) Fingerprint(ctx context.Context) (<-chan *drivers.Fingerprint, error) {
	ch := make(chan *drivers.Fingerprint)
	go d.handleFingerprint(ctx, ch)
	return ch, nil
}

func (d *DriverPlugin) Shutdown(ctx context.Context) error {
	d.signalShutdown()
	return nil
}

func (d *DriverPlugin) handleFingerprint(ctx context.Context, ch chan<- *drivers.Fingerprint) {
	defer close(ch)
	ticker := time.NewTimer(0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.ctx.Done():
			d.logger.Debug("closing fingerprint worker")
			return
		case <-ticker.C:
			ticker.Reset(fingerprintPeriod)
			ch <- d.buildFingerprint()
		}
	}
}

func (d *DriverPlugin) buildFingerprint() *drivers.Fingerprint {
	var health drivers.HealthState
	var desc string
	attrs := make(map[string]*pstructs.Attribute)
	attrs["driver.damon.version"] = pstructs.NewStringAttribute(version.Number)
	attrs["driver.damon.enforce_cpu_limit"] = pstructs.NewBoolAttribute(d.config.EnforceCPULimit)
	attrs["driver.damon.enforce_memory_limit"] = pstructs.NewBoolAttribute(d.config.EnforceMemoryLimit)
	attrs["driver.damon.restricted_token"] = pstructs.NewBoolAttribute(d.config.RestrictedToken)
	if d.config.Enabled {
		health = drivers.HealthStateHealthy
		desc = "ready"
	} else {
		health = drivers.HealthStateUndetected
		desc = "disabled"

	}
	return &drivers.Fingerprint{
		Attributes:        attrs,
		Health:            health,
		HealthDescription: desc,
	}
}

func (d *DriverPlugin) StartTask(tc *drivers.TaskConfig) (*drivers.TaskHandle, *drivers.DriverNetwork, error) {
	if _, ok := d.taskStore.Get(tc.ID); ok {
		return nil, nil, fmt.Errorf("task with ID '%s' is already running", tc.ID)
	}
	var driverCfg TaskConfig
	if err := tc.DecodeDriverConfig(&driverCfg); err != nil {
		return nil, nil, fmt.Errorf("unable to decode driver config: %v", err)
	}
	d.logger.Info("starting damon task", "driver_cfg", log.Fmt("%+v", driverCfg), "task_cfg", log.Fmt("%+v", tc))
	handle := drivers.NewTaskHandle(taskHandleVersion)
	handle.Config = tc

	dexe, err := newDamonExec(tc, driverCfg, d.logger)
	if err != nil {
		return nil, nil, err
	}

	th, err := dexe.startContainer(tc)
	if err != nil {
		return nil, nil, err
	}
	d.taskStore.Put(tc.ID, th)
	go th.run()

	return handle, nil, nil
}

func (d *DriverPlugin) RecoverTask(th *drivers.TaskHandle) error {
	if th == nil {
		return fmt.Errorf("handle cannot be nil")
	}
	if _, ok := d.taskStore.Get(th.Config.ID); ok {
		d.logger.Trace("nothing to recover; task already exists",
			"task_id", th.Config.ID,
			"task_name", th.Config.Name,
		)
		return nil
	}
	_, _, err := d.StartTask(th.Config)
	return err
}

func (d *DriverPlugin) WaitTask(ctx context.Context, taskID string) (<-chan *drivers.ExitResult, error) {
	handle, ok := d.taskStore.Get(taskID)
	if !ok {
		return nil, drivers.ErrTaskNotFound
	}
	ch := make(chan *drivers.ExitResult)
	go d.handleWait(ctx, handle, ch)
	return ch, nil
}

func (d *DriverPlugin) handleWait(ctx context.Context, handle *taskHandle, ch chan *drivers.ExitResult) {
	defer close(ch)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.ctx.Done():
			d.logger.Debug("closing handleWait worker", "pid", handle.pid)
			return
		case <-ticker.C:
			s := handle.TaskStatus()
			if s.State == drivers.TaskStateExited {
				ch <- handle.exitResult
			}
		}
	}
}

func (d *DriverPlugin) StopTask(taskID string, timeout time.Duration, signal string) error {
	handle, ok := d.taskStore.Get(taskID)
	if !ok {
		return drivers.ErrTaskNotFound
	}
	if err := handle.shutdown(timeout); err != nil {
		return fmt.Errorf("executor Shutdown failed: %v", err)
	}
	return nil
}

func (d *DriverPlugin) DestroyTask(taskID string, force bool) error {
	handle, ok := d.taskStore.Get(taskID)
	if !ok {
		return drivers.ErrTaskNotFound
	}

	if handle.IsRunning() && !force {
		return fmt.Errorf("cannot destroy running task")
	}

	if handle.IsRunning() {
		// grace period is chosen arbitrary here
		if err := handle.shutdown(1 * time.Minute); err != nil {
			handle.logger.Error("failed to destroy executor", "err", err)
		}
	}
	d.logger.Debug("destroyed task", "task_id", taskID)
	d.taskStore.Delete(taskID)
	return nil
}
func (d *DriverPlugin) InspectTask(taskID string) (*drivers.TaskStatus, error) {
	handle, ok := d.taskStore.Get(taskID)
	if !ok {
		return nil, drivers.ErrTaskNotFound
	}
	return handle.TaskStatus(), nil

}
func (d *DriverPlugin) TaskStats(ctx context.Context, taskID string, interval time.Duration) (<-chan *cstructs.TaskResourceUsage, error) {
	handle, ok := d.taskStore.Get(taskID)
	if !ok {
		return nil, drivers.ErrTaskNotFound
	}
	return handle.stats(ctx, interval)
}
func (d *DriverPlugin) TaskEvents(ctx context.Context) (<-chan *drivers.TaskEvent, error) {
	return d.eventer.TaskEvents(ctx)
}

func (d *DriverPlugin) SignalTask(taskID string, signal string) error {
	handle, ok := d.taskStore.Get(taskID)
	if !ok {
		return drivers.ErrTaskNotFound
	}
	switch signal {
	// special case for killing the process
	case "SIGINT", "SIGKILL":
		return handle.container.Shutdown(30 * time.Second)
	// Not sure if any of these signals actually work
	default:
		sig, ok := SignalLookup[signal]
		if !ok {
			sig = os.Kill
			d.logger.Warn("unknown signal to send to task, using SIGINT instead", "signal", signal, "task_id", handle.taskConfig.ID)
		}
		return handle.container.Signal(sig)
	}
}
func (d *DriverPlugin) ExecTask(taskID string, cmd []string, timeout time.Duration) (*drivers.ExecTaskResult, error) {
	handle, ok := d.taskStore.Get(taskID)
	if !ok {
		return nil, drivers.ErrTaskNotFound
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	task, err := handle.container.Exec(container.TaskConfig{
		EnvList: handle.taskConfig.EnvList(),
		Dir:     handle.taskConfig.TaskDir().Dir,
		Stderr:  stderr,
		Stdout:  stdout,
	})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	res, err := task.Wait(ctx)
	return &drivers.ExecTaskResult{
		ExitResult: &drivers.ExitResult{
			Err:      err,
			ExitCode: res,
		},
		Stdout: stdout.Bytes(),
		Stderr: stderr.Bytes(),
	}, nil
}
