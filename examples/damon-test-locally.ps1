#!powershell

###
# Set up some environment variable to simulate a Nomad Environment
###


$env:DAMON_LOG_MAX_SIZE=100
$env:DAMON_LOG_MAX_FILES=3
$env:DAMON_LOG_DIR=$pwd.Path
$env:DAMON_LOG_NAME="damon.log"

$env:NOMAD_ALLOC_DIR = $pwd.Path
$env:NOMAD_ALLOC_ID  = "3f07eca7-adbc-454c-9e97-06d78ff28e28"
$env:NOMAD_ALLOC_INDEX = "1"
$env:NOMAD_TASK_NAME = "damon-test-task"
$env:NOMAD_GROUP_NAME ="damon-group"
$env:NOMAD_JOB_NAME = "damon-job"
$env:NOMAD_DC="dev"
$env:NOMAD_REGION="eastus2"
$env:NOMAD_CPU_LIMIT="1024"
$env:NOMAD_MEMORY_LIMIT="1024"

$env:DAMON_ENFORCE_CPU_LIMIT="Y"
$env:DAMON_CPU_LIMIT=2048

$env:DAMON_ENFORCE_MEMORY_LIMIT="Y"
$env:DAMON_MEMORY_LIMIT=128

$env:DAMON_RESTRICTED_TOKEN = "Y"

$env:DAMON_ADDR = "localhost:8080"
$env:DAMON_METRICS_ENDPOINT = "/metrics"

###
# Run Damon with your executable
###

# Replace with location of your actual executables
# or put them both in your PATH
& damon.exe myservice.exe