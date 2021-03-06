package app

import (
	"time"

	"github.com/projecteru/eru-agent/common"
	"github.com/projecteru/eru-agent/g"
	"github.com/projecteru/eru-agent/logs"
	"github.com/projecteru/eru-metric/metric"
)

func Metric() {
	metric.SetGlobalSetting(
		g.Docker, time.Duration(common.STATS_TIMEOUT),
		time.Duration(common.STATS_FORCE_DONE),
		common.VLAN_PREFIX, common.DEFAULT_BR,
	)
	logs.Info("Metrics initiated")
}

func (self *EruApp) Report() {
	t := time.NewTicker(self.Step)
	defer t.Stop()
	defer self.Client.Close()
	defer logs.Info(self.Name, self.EntryPoint, self.ID[:12], "metrics report stop")
	logs.Info(self.Name, self.EntryPoint, self.ID[:12], "metrics report start")
	for {
		select {
		case now := <-t.C:
			go func() {
				if info, err := self.UpdateStats(self.ID); err == nil {
					if isLimit {
						limitChan <- SoftLimit{self.ID, info}
					}
					rate := self.CalcRate(info, now)
					self.SaveLast(info)
					go self.Send(rate)
				} else {
					logs.Info("Update mertic failed", self.ID[:12])
				}
			}()
		case <-self.Stop:
			return
		}
	}
}
