# Damon

Damon is a supervisor program to constrain windows executables that are run under the `raw_exec` driver in Nomad.

## Usage

To use Damon, run it before your command.

```
damon.exe yourapp.exe [args]
```

## Configuration

Damon uses environment variables to configure process monitoring and resource constraints.

### Logging Options

- `DAMON_LOG_MAX_FILES`: the number of old logs to keep after rotating.
- `DAMON_LOG_MAX_SIZE`: the maximum size (in MB) of the active log file before it gets rotated.
- `DAMON_LOG_DIR`: directory in which to place damon log files. When `DAMON_LOG_DIR` is unset, it will attempt to use the standard nomad log directory `${NOMAD_ALLOC_DIR}/logs`. If `NOMAD_ALLOC_DIR` is unset, then it will default to the current working directory.
- `DAMON_NOMAD_LOG_SUFFIX`: Is appended to the log name of the active log file. Rotated log files contain a datestamp. The default value is `.damon.log`
- `DAMON_LOG_NAME`: Is the full name of the log file (without the directory) - Setting this overrides `DAMON_NOMAD_LOG_SUFFIX`. When this is unset, it will default to `${NOMAD_TASK_NAME}${DAMON_NOMAD_LOG_SUFFIX}`

### Constraint Options

- `DAMON_ENFORCE_CPU_LIMIT`: When set to `Y` - it enforces CPU constraints on the wrapped process.
- `DAMON_ENFORCE_MEMORY_LIMIT`: When set to `Y` - it enforces memory limits on the wrapped process.
- `DAMON_CPU_LIMIT`: The CPU Limit in MHz. Defaults to `NOMAD_CPU_LIMIT`.
- `DAMON_MEMORY_LIMIT`: The Memory Limit in MB. Defaults to `NOMAD_MEMORY_LIMIT`.
- `DAMON_RESTRICTED_TOKEN`: When set to `Y` - it runs the wrapped process with a [Restricted Token](https://docs.microsoft.com/en-us/windows/desktop/SecAuthZ/restricted-tokens):
    - Drops all [Privileges](https://docs.microsoft.com/en-us/windows/desktop/secauthz/privileges)
    - Disables the `BUILTIN\Administrator` SID

### Metrics Options

- `DAMON_ADDR`: Listens on this address to serve prometheus metrics. Default: `${NOMAD_ADDR_damon}`
    This option is designed to work with the `NOMAD_ADDR_damon` environment variable.
    This means you should change your job spec to:
    - request a port labeled `"damon"`
    - add a service to the task that advertises the "damon" port to Consul service discovery - so that your prometheus infrastructure can find it and scrape it.
- `DAMON_METRICS_ENDPOINT`: The path to the prometheus metrics endpoint. Default: `/metrics`

## Building & Testing Damon

Included with this repository is `make.ps1` which can be used to build `damon.exe` and also run tests.

### Build Binary

```posh
.\make.ps1 -Build
```

### Lint Code

Runs [golangci-lint](https://github.com/golangci/golangci-lint) against the codebase. It will [Install golangci-lint](https://github.com/golangci/golangci-lint#local-installation) if it doesn't exist in `${GOPATH}/bin`.

```posh
.\make.ps1 -Lint
```

### Test Code

Runs tests and generates code coverage files.

```posh
.\make.ps1 -Test
```

## Give it a Try

Check out the [examples directory](examples/) for scripts and job definitions.

- Locally: [damon-test-locally.ps1](examples/damon-test-locally.ps1)
- On Nomad: [damon-job.nomad](examples/damon-job.nomad)

Be sure to alter to environment variables, artifact locations, etc... to match your environment.
