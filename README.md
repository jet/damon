# Damon 

Damon is a supervisor program to constrain windows executables that are run under the `raw_exec` driver.

## Usage

To use Damon, run it before your command.

```
damon.exe yourapp.exe [args]
```

## Configuration

Damon uses environment variables to configure process monitoring and resource constraints. It uses a combination of `NOMAD_*` and `DAMON_*` environment variables. `DAMON_*` has higher precendece relative to `NOMAD_*`.

