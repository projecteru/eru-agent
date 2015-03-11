package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/keimoon/gore"

	"./common"
	"./defines"
	"./lenz"
	"./logs"
	"./metrics"
	"./utils"
)

var Lenz *lenz.LenzForwarder
var Metrics *metrics.MetricsRecorder

var Status *StatusMoniter

func main() {
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

	common.Docker = defines.NewDocker(config.Docker.Endpoint)

	Lenz = lenz.NewLenz(config.Lenz)
	cleaner := lenz.NewCleaner(config.Cleaner)
	Metrics = metrics.NewMetricsRecorder(config.HostName, config.Metrics)

	utils.WritePid(config.PidFile)
	defer os.Remove(config.PidFile)

	Status = NewStatus()
	Status.Load()
	go Status.Listen()
	go Metrics.Report()
	go cleaner.Clean()

	var c = make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGHUP)
	signal.Notify(c, syscall.SIGKILL)
	signal.Notify(c, syscall.SIGQUIT)
	logs.Info("Catch", <-c)
	Metrics.Stop()
	cleaner.Stop()
}
