package app

import (
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
)

type SoftLimit struct {
	flag      bool
	cid       string
	mem_usage float64
}

var limitChan chan SoftLimit = make(chan SoftLimit)
var usage map[string]float64 = make(map[string]float64)

func Limit() {
	go calcMemoryUsage()
}

func calcMemoryUsage() {
	for {
		select {
		case d := <-limitChan:
			if !d.flag {
				logs.Info("Get mem stats failed", d.cid)
			}
			usage[d.cid] = d.mem_usage
			if len(usage) == len(Apps) {
				judgeMemoryUsage()
				cleanUsage()
			}
		}
	}
}

func judgeMemoryUsage() {
	var totalUsage float64 = 0.0
	var rate map[string]float64 = make(map[string]float64)
	for k, u := range usage {
		totalUsage = totalUsage + u
		//TODO ugly
		if _, ok := Apps[k].Extend["__memory__"]; !ok {
			rate[k] = 0.0
			continue
		}
		v, _ := Apps[k].Extend["__memory__"].(float64)
		rate[k] = u / v
	}
	logs.Debug("Current memory usage", totalUsage, g.Config.Docker.Memlimit)
	if totalUsage < g.Config.Docker.Memlimit {
		return
	}
	logs.Info("Current memory warning", totalUsage, g.Config.Docker.Memlimit)
	for {
		if totalUsage < g.Config.Docker.Memlimit {
			return
		}
		var max float64 = 0
		var cid string = ""
		for k, _ := range rate {
			if max < usage[k] {
				max = usage[k]
				cid = k
			}
		}
		softOOMKill(cid, usage[cid])
		totalUsage -= max
		delete(rate, cid)
	}
}

func cleanUsage() {
	for k, _ := range usage {
		delete(usage, k)
	}
}

func softOOMKill(cid string, rate float64) {
	logs.Info(cid, rate, "oom killed")
}
