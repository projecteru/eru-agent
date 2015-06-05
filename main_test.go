package main

import (
	"./common"
	"./defines"
	"./lenz"
	"./metrics"
)

func InitTest() {
	load("agent.yaml")
	common.Docker = defines.NewDocker(config.Docker.Endpoint)
	defines.MockDocker(common.Docker)
	if Status == nil {
		Status = NewStatus()
	}
	if Lenz == nil {
		Lenz = lenz.NewLenz(config.Lenz)
	}
	if metrics.Metrics == nil {
		metrics.Metrics = metrics.NewMetricsRecorder(config.HostName, config.Metrics)
	}
}
