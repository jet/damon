package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"

	"github.com/jet/damon/container"
	"github.com/jet/damon/log"
	"github.com/jet/damon/metrics"
	"github.com/jet/damon/version"
	"github.com/jet/damon/win32"
)

func main() {
	// Limit Damon to 1 CPU
	runtime.GOMAXPROCS(1)
	vinfo := version.GetInfo()

	if len(os.Args) < 2 {
		// print version and exit - no args
		fmt.Println(vinfo.FullString(true))
		os.Exit(0)
	}

	var cmd *exec.Cmd
	if len(os.Args) > 2 {
		cmd = exec.Command(os.Args[1], os.Args[2:]...)
	} else {
		cmd = exec.Command(os.Args[1])
	}

	lcfg := LogConfigFromEnvironment()
	fields := NomadLogFields()
	logger, err := log.NewLogger(lcfg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	logger = logger.WithFields(fields)
	logger.WithFields(map[string]interface{}{
		"version":  vinfo,
		"revision": version.GitCommit,
		"cmdline":  os.Args,
	}).Logln("damon starting")
	clogger := logger.WithFields(map[string]interface{}{
		"version":  vinfo.String(),
		"revision": version.GitCommit,
		"cmdline":  os.Args,
	})
	ccfg, err := LoadContainerConfigFromEnvironment()
	if err != nil {
		logger.Error(err, "unable to load container configuration from environment variables")
	}
	win32.SetLogger(logger)
	resources := win32.GetSystemResources()
	labels := make(map[string]string)
	for k, v := range fields {
		labels[k] = fmt.Sprintf("%v", v)
	}
	m := metrics.Metrics{
		Cores:      resources.CPUNumCores,
		MHzPerCore: resources.CPUMhzPercore,
		Namespace:  "damon",
		Labels:     labels,
	}
	m.Init()
	c := container.Container{
		Command: cmd,
		Config:  ccfg,
		Logger:  clogger,
		OnStats: func(s container.ProcessStats) {
			m.OnStats(s)
		},
		OnViolation: func(v container.LimitViolation) {
			m.OnViolation(v)
		},
	}
	if err := c.Start(); err != nil {
		logger.Error(err, "damon startup error")
		os.Exit(1)
	}
	exitCh := make(chan struct{})
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh)
	go func() {
		<-sigCh
		close(exitCh)
	}()
	if addr := ListenAddress(); addr != "" {
		go func() {
			endpoint := MetricsEndpoint()
			mux := http.NewServeMux()
			mux.Handle(endpoint, m.Handler())
			srv := &http.Server{
				Addr: addr,
				Handler: mux,
			}
			logger.Logf("metrics on http://%s/%s",addr,endpoint)
			logger.Error(srv.ListenAndServe(),"error closing http server")
		}()
	}
	pr, err := c.Wait(exitCh)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"version":  vinfo,
			"revision": version.GitCommit,
			"cmdline":  os.Args,
		}).Error(err, "process exited with an error")
	}

	logger.WithFields(map[string]interface{}{
		"version":     vinfo,
		"revision":    version.GitCommit,
		"cmdline":     os.Args,
		"start":       pr.Start,
		"end":         pr.End,
		"run_time":    pr.End.Sub(pr.Start),
		"exit_status": pr.ExitCode,
	}).Logln("damon exiting")
	os.Exit(pr.ExitCode)
}
