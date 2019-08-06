package plugin

import (
	"time"

	"github.com/hashicorp/nomad/plugins/drivers"
)

type TaskState struct {
	PID        int
	Start      time.Time
	JobName    string
	TaskConfig *drivers.TaskConfig
}

func (t *TaskState) Recover(cfg TaskConfig) (*taskHandle, error) {
	
	return nil, nil
}
