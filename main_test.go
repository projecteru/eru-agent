package main

import (
	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/lenz"
	"github.com/HunanTV/eru-agent/metrics"
)

func InitTest() {
	load("agent.yaml")
	common.Docker, _ = defines.NewDocker(config.Docker.Endpoint)
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
