package app

import (
	"fmt"
	"time"

	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/HunanTV/eru-metric/metric"
	"github.com/fsouza/go-dockerclient"
)

func (self *EruApp) InitMetric() bool {
	dockerAddr := g.Config.Docker.Endpoint
	cert := g.Config.Docker.Cert
	key := g.Config.Docker.Key
	ca := g.Config.Docker.Ca
	dockerclient, _ := docker.NewTLSClient(dockerAddr, cert, key, ca)
	metric.SetGlobalSetting(
		dockerclient,
		time.Duration(g.Config.Metrics.Timeout)*time.Millisecond,
		time.Duration(g.Config.Metrics.Force)*time.Millisecond,
		common.VLAN_PREFIX,
		common.DEFAULT_BR)
	return true
}

func (self *EruApp) Report() {
	defer logs.Info(self.Name, self.EntryPoint, self.ID[:12], "metrics report stop")
	logs.Info(self.Name, self.EntryPoint, self.ID[:12], "metrics report start")

	serv := metric.CreateMetric(self.Metric.Step, self.Client, self.Tag, self.Endpoint)

	for {
		select {
		case now := <-time.Tick(serv.Step):
			go func() {
				if info, err := serv.UpdateStats(self.Extend["cid"].(string)); err == nil {
					fmt.Println(info)
					rate := serv.CalcRate(info, now)
					serv.SaveLast(info)
					go serv.Send(rate)
				}
			}()
		case <-serv.Stop:
			return
		}
	}

}

func (self *EruApp) Exit() {
	serv := metric.CreateMetric(self.Metric.Step, self.Client, self.Tag, self.Endpoint)
	serv.Exit()
}
