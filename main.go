package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/keimoon/gore"

	"github.com/HunanTV/eru-agent/api"
	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/lenz"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/HunanTV/eru-agent/metrics"
	"github.com/HunanTV/eru-agent/status"
	"github.com/HunanTV/eru-agent/utils"
)

var VLan *VLanSetter

func main() {
	var err error
	LoadConfig()

	common.Rds = &gore.Pool{
		InitialConn: config.Redis.Min,
		MaximumConn: config.Redis.Max,
	}

	redisHost := fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port)
	if err := common.Rds.Dial(redisHost); err != nil {
		logs.Assert(err, "Redis init failed")
	}
	defer common.Rds.Close()

	if common.Docker, err = defines.NewDocker(
		config.Docker.Endpoint,
		config.Docker.Cert,
		config.Docker.Key,
		config.Docker.Ca,
	); err != nil {
		logs.Assert(err, "Docker")
	}

	lenz.Lenz = lenz.NewLenz(config.Lenz)
	metrics.Metrics = metrics.NewMetricsRecorder(config.HostName, config.Metrics)
	status.Status = status.NewStatus(config.HostName, config.Eru.Endpoint)

	utils.WritePid(config.PidFile)
	defer os.Remove(config.PidFile)

	VLan = NewVLanSetter()
	go VLan.Watcher()
	status.Status.Load()
	go status.Status.Watcher()
	go status.Status.Listen()
	go Ping()

	if config.API.Http {
		go api.HTTPServe(config.API)
	}

	var c = make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGHUP)
	signal.Notify(c, syscall.SIGKILL)
	signal.Notify(c, syscall.SIGQUIT)
	logs.Info("Catch", <-c)
}
