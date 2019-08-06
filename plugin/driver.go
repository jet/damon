package plugin

import (
	"context"
	"time"

	log "github.com/hashicorp/go-hclog"
	cstructs "github.com/hashicorp/nomad/client/structs"
	"github.com/hashicorp/nomad/plugins/base"
	"github.com/hashicorp/nomad/plugins/drivers"
	"github.com/hashicorp/nomad/plugins/shared/hclspec"
	pstructs "github.com/hashicorp/nomad/plugins/shared/structs"
	"github.com/jet/damon/version"
)

func NewDriverPlugin(log log.Logger) *DriverPlugin {
	return &DriverPlugin{log: log}
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
		Type:          base.PluginTypeBase,
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
		SendSignals: false,
		Exec:        false,
		FSIsolation: drivers.FSIsolationNone,
	}
)

type DriverPlugin struct {
	log         log.Logger
	ctx         context.Context
	config      *Config
	nomadConfig *base.ClientDriverConfig
	logger      log.Logger
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
	return nil, nil
}

func (d *DriverPlugin) handleFingerprint(ctx context.Context, ch chan<- *drivers.Fingerprint) {
	defer close(ch)
	ticker := time.NewTimer(0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.ctx.Done():
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

func (d *DriverPlugin) RecoverTask(th *drivers.TaskHandle) error {
	return nil
}
func (d *DriverPlugin) StartTask(tc *drivers.TaskConfig) (*drivers.TaskHandle, *drivers.DriverNetwork, error) {
	return nil, nil, nil
}
func (d *DriverPlugin) WaitTask(ctx context.Context, taskID string) (<-chan *drivers.ExitResult, error) {
	return nil, nil
}
func (d *DriverPlugin) StopTask(taskID string, timeout time.Duration, signal string) error {
	return nil
}
func (d *DriverPlugin) DestroyTask(taskID string, force bool) error {
	return nil
}
func (d *DriverPlugin) InspectTask(taskID string) (*drivers.TaskStatus, error) {
	return nil, nil
}
func (d *DriverPlugin) TaskStats(ctx context.Context, taskID string, interval time.Duration) (<-chan *cstructs.TaskResourceUsage, error) {
	return nil, nil
}
func (d *DriverPlugin) TaskEvents(ctx context.Context) (<-chan *drivers.TaskEvent, error) {
	return nil, nil
}

func (d *DriverPlugin) SignalTask(taskID string, signal string) error {
	return nil
}
func (d *DriverPlugin) ExecTask(taskID string, cmd []string, timeout time.Duration) (*drivers.ExecTaskResult, error) {
	return nil, nil
}
