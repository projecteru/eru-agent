package main

import (
	"os"
	"os/signal"
	"syscall"

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
