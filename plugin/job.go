package plugin

import (
	"fmt"
	"io"

	log "github.com/hashicorp/go-hclog"
	"github.com/jet/damon/win32"
)

func wrapJob(logger log.Logger) (io.Closer, error) {
	cp, err := win32.CurrentProcess()
	if err != nil {
		logger.Error("error getting plugin process", "error", err)
		return nil, err
	}
	jo, err := win32.CreateJobObject(fmt.Sprintf("damon-%d", cp.Pid()))
	if err != nil {
		logger.Error("error creating job object for plugin process", "error", err)
		return nil, err
	}
	if err := jo.SetInformation(&win32.ExtendedLimitInformation{
		KillOnJobClose: true,
	}); err != nil {
		logger.Error("error setting job object info on plugin process", "error", err)
		return nil, err
	}
	if err := jo.Assign(cp); err != nil {
		logger.Error("error assigning job object to plugin process", "error", err)
		return nil, err
	}
	return jo, nil
}
