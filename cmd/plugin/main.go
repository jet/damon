package main

import (
	log "github.com/hashicorp/go-hclog"

	"github.com/hashicorp/nomad/plugins"
	"github.com/jet/damon/plugin"
)

func main() {
	// Serve the plugin
	plugins.Serve(factory)
}

// factory returns a new instance of the LXC driver plugin
func factory(log log.Logger) interface{} {
	return plugin.NewDriverPlugin(log)
}
