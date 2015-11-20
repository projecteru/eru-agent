package app

import (
	"fmt"

	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/keimoon/gore"
)

type SoftLimit struct {
	cid  string
	info map[string]uint64
}

var limitChan chan SoftLimit = make(chan SoftLimit)
var usage map[string]uint64 = make(map[string]uint64)
var isLimit bool = false

func Limit() {
	if g.Config.Limit.Memory != 0 {
		logs.Info("App memory soft limit start")
		isLimit = true
		go calcMemoryUsage()
	}
}

func calcMemoryUsage() {
	for {
		select {
		case d := <-limitChan:
			if v, ok := d.info["mem_usage"]; ok {
				usage[d.cid] = v
			} else {
				usage[d.cid] = 0
			}
			var doCalc bool = true
			for id, _ := range Apps {
				if _, ok := usage[id]; !ok {
					doCalc = false
					break
				}
			}
			if doCalc {
				judgeMemoryUsage()
			}
		}
	}
}

func judgeMemoryUsage() {
	var totalUsage uint64 = 0
	var rate map[string]float64 = make(map[string]float64)
	for cid, usage := range usage {
		totalUsage = totalUsage + usage
		//TODO ugly
		if v, ok := Apps[cid].Extend["__memory__"]; !ok {
			rate[cid] = 0.0
			continue
		} else {
			define, _ := v.(float64)
			rate[cid] = float64(usage) / define
		}
	}
	logs.Debug("Current memory usage", totalUsage, "max", g.Config.Limit.Memory)
	for {
		if totalUsage < g.Config.Limit.Memory {
			return
		}
		var exceedRate float64 = 0.0
		var cid string = ""
		for k, v := range rate {
			if exceedRate >= v {
				continue
			}
			exceedRate = v
			cid = k
		}
		if cid == "" {
			logs.Info("MemLimit can not stop containers")
			break
		}
		softOOMKill(cid, exceedRate)
		totalUsage -= usage[cid]
		delete(rate, cid)
	}
	for k, _ := range usage {
		delete(usage, k)
	}
}

func softOOMKill(cid string, rate float64) {
	logs.Debug("OOM killed", cid[:12])
	conn := g.GetRedisConn()
	defer g.ReleaseRedisConn(conn)

	key := fmt.Sprintf("eru:agent:%s:container:reason", cid)
	if _, err := gore.NewCommand("SET", key, common.OOM_KILLED).Run(conn); err != nil {
		logs.Info("OOM killed set flag", err)
	}
	if err := g.Docker.StopContainer(cid, 10); err != nil {
		logs.Info("OOM killed failed", cid[:12])
		return
	}
	logs.Info("OOM killed success", cid[:12])
}
