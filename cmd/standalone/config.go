package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/jet/damon/container"
	"github.com/jet/damon/log"
)

const DefaultLogMaxSizeMB = 10
const DefaultLogMaxFiles = 5
const DefaultMetricsEndpoint = "/metrics"

const (
	EnvDamonContainerName  = "DAMON_CONTAINER_NAME"
	EnvDamonLogMaxSizeMB   = "DAMON_LOG_MAX_SIZE"
	EnvDamonLogMaxFiles    = "DAMON_LOG_MAX_FILES"
	EnvDamonLogDir         = "DAMON_LOG_DIR"
	EnvDamonLogName        = "DAMON_LOG_NAME"
	EnvDamonNomadLogSuffix = "DAMON_NOMAD_LOG_SUFFIX"
	EnvNomadAllocDir       = "NOMAD_ALLOC_DIR"
	EnvNomadTaskName       = "NOMAD_TASK_NAME"
	EnvNomadAllocID        = "NOMAD_ALLOC_ID"
	EnvNomadAllocIndex     = "NOMAD_ALLOC_INDEX"
	EnvNomadGroupName      = "NOMAD_GROUP_NAME"
	EnvNomadJobName        = "NOMAD_JOB_NAME"
	EnvNomadDC             = "NOMAD_DC"
	EnvNomadRegion         = "NOMAD_REGION"
	EnvNomadDamonAddress   = "NOMAD_ADDR_damon"

	EnvDamonEnforceCPULimit    = "DAMON_ENFORCE_CPU_LIMIT"
	EnvDamonEnforceMemoryLimit = "DAMON_ENFORCE_MEMORY_LIMIT"
	EnvDamonCPULimit           = "DAMON_CPU_LIMIT"
	EnvNomadCPULimit           = "NOMAD_CPU_LIMIT"
	EnvDamonMemoryLimit        = "DAMON_MEMORY_LIMIT"
	EnvNomadMemoryLimit        = "NOMAD_MEMORY_LIMIT"
	EnvDamonRestrictedToken    = "DAMON_RESTRICTED_TOKEN"
	EnvDamonAddress            = "DAMON_ADDR"
	EnvDamonMetricsEndpoint    = "DAMON_METRICS_ENDPOINT"
)

func LogConfigFromEnvironment() log.LogConfig {
	cfg := log.LogConfig{
		LogDir:         os.Getenv(EnvDamonLogDir),
		LogName:        os.Getenv(EnvDamonLogName),
		NomadAllocDir:  os.Getenv(EnvNomadAllocDir),
		NomadLogSuffix: os.Getenv(EnvDamonNomadLogSuffix),
		NomadTaskName:  os.Getenv(EnvNomadTaskName),
		MaxLogFiles:    DefaultLogMaxFiles,
		MaxSizeMB:      DefaultLogMaxSizeMB,
	}
	if sz := os.Getenv(EnvDamonLogMaxSizeMB); sz != "" {
		if i, err := strconv.ParseInt(sz, 10, 64); err == nil {
			cfg.MaxSizeMB = int(i)
		}
	}

	if sz := os.Getenv(EnvDamonLogMaxFiles); sz != "" {
		if i, err := strconv.ParseInt(sz, 10, 64); err == nil {
			cfg.MaxLogFiles = int(i)
		}
	}
	return cfg
}

var nomadEnvToFields = map[string]string{
	EnvNomadDC:         "nomad_dc",
	EnvNomadRegion:     "nomad_region",
	EnvNomadJobName:    "nomad_job_name",
	EnvNomadGroupName:  "nomad_group_name",
	EnvNomadTaskName:   "nomad_task_name",
	EnvNomadAllocID:    "nomad_alloc_id",
	EnvNomadAllocIndex: "nomad_alloc_index",
}

var nomadFieldsOnce sync.Once
var nomadFields map[string]interface{}

func NomadLogFields() map[string]interface{} {
	nomadFieldsOnce.Do(func() {
		nomadFields = make(map[string]interface{})
		for env, field := range nomadEnvToFields {
			if v := os.Getenv(env); v != "" {
				nomadFields[field] = v
			}
		}
	})
	return nomadFields
}

func envToBool(env string, def bool) bool {
	if env := os.Getenv(env); env != "" {
		switch strings.ToLower(strings.TrimSpace(env)) {
		case "y", "yes", "true", "1":
			return true
		case "n", "no", "false", "0":
			return false
		}
	}
	return def
}

func envStr(def string, envs ...string) string {
	for _, e := range envs {
		if env := os.Getenv(e); env != "" {
			return env
		}
	}
	return def
}

func envToInt(def int64, envs ...string) (int64, error) {
	for _, e := range envs {
		if env := os.Getenv(e); env != "" {
			i, err := strconv.ParseInt(env, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("error parsing environment %s=%s as integer: %v", e, env, err)
			}
			return int64(i), nil
		}
	}
	return def, nil
}

func ListenAddress() string {
	if env := os.Getenv(EnvDamonAddress); env != "" {
		return env
	}
	if env := os.Getenv(EnvNomadDamonAddress); env != "" {
		return env
	}
	return ""
}

func MetricsEndpoint() string {
	if env := os.Getenv(EnvDamonMetricsEndpoint); env != "" {
		return env
	}
	return DefaultMetricsEndpoint
}

func nomadContainerName() string {
	if alloc, name := os.Getenv(EnvNomadAllocID), os.Getenv(EnvNomadTaskName); alloc != "" && name != "" {
		return fmt.Sprintf("damon:%s.%s", alloc, name)
	}
	return ""
}

func LoadContainerConfigFromEnvironment() (container.Config, error) {
	var cfg container.Config
	if env := os.Getenv(EnvDamonContainerName); env != "" {
		cfg.Name = envStr("", EnvDamonContainerName, nomadContainerName())
	}
	cpu, err := envToInt(0, EnvDamonCPULimit, EnvNomadCPULimit)
	if err != nil {
		return cfg, err
	}
	if cpu > 0 {
		cfg.EnforceCPU = envToBool(EnvDamonEnforceCPULimit, true)
		cfg.CPUMHzLimit = int(cpu)
	}
	mem, err := envToInt(0, EnvDamonMemoryLimit, EnvNomadMemoryLimit)
	if err != nil {
		return cfg, err
	}
	if mem > 0 {
		cfg.EnforceMemory = envToBool(EnvDamonEnforceMemoryLimit, true)
		cfg.MemoryMBLimit = int(mem)
	}
	cfg.RestrictedToken = envToBool(EnvDamonRestrictedToken, false)

	if cfg.EnforceCPU && cfg.CPUMHzLimit < container.MinimumCPUMHz {
		return cfg, errors.Errorf("CPU limit is too low. Minimum CPU MHz is %d - got %d", container.MinimumCPUMHz, cfg.CPUMHzLimit)
	}
	return cfg, nil
}
