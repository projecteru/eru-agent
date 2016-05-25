package app

import (
	"time"

	"github.com/projecteru/eru-agent/common"
	"github.com/projecteru/eru-agent/g"
	"github.com/projecteru/eru-metric/metric"

	log "github.com/Sirupsen/logrus"
)

func Metric() {
	metric.SetGlobalSetting(
		g.Docker, time.Duration(common.STATS_TIMEOUT),
		time.Duration(common.STATS_FORCE_DONE),
		common.VLAN_PREFIX, common.DEFAULT_BR,
	)
	log.Info("Metrics initiated")
}

func (self *EruApp) Report() {
	t := time.NewTicker(self.Step)
	defer t.Stop()
	defer self.Client.Close()
	defer log.Infof("%s %s %s metrics report stop", self.Name, self.EntryPoint, self.ID[:12])
	log.Infof("%s %s %s metrics report start", self.Name, self.EntryPoint, self.ID[:12])
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
					log.Infof("Update mertic failed %s", self.ID[:12])
				}
			}()
		case <-self.Stop:
			return
		}
	}
}
