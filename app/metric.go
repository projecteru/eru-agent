package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/fsouza/go-dockerclient"
	"github.com/open-falcon/common/model"
)

func (self *EruApp) InitMetric() bool {
	if !self.updateStats() {
		logs.Info("Init mertics failed", self.Meta.ID[:12])
		return false
	}
	self.Last = time.Now()
	self.saveLast()
	return true
}

func (self *EruApp) Exit() {
	self.Stop <- true
	close(self.Stop)
}

func (self *EruApp) Report() {
	defer self.Client.Close()
	defer logs.Info(self.Name, self.EntryPoint, self.ID[:12], "metrics report stop")
	logs.Info(self.Name, self.EntryPoint, self.ID[:12], "metrics report start")
	for {
		select {
		case now := <-time.Tick(self.Step):
			go func() {
				upOk := self.updateStats()
				if isLimit {
					limitChan <- SoftLimit{upOk, self.ID, self.Info}
				}
				if !upOk {
					logs.Info("Update mertic failed", self.Meta.ID[:12])
					return
				}
				self.calcRate(now)
				// for safe
				go self.send(self.Rate)
			}()
		case <-self.Stop:
			return
		}
	}
}

func (self *EruApp) updateStats() bool {
	statsChan := make(chan *docker.Stats)
	doneChan := make(chan bool)
	opt := docker.StatsOptions{self.ID, statsChan, false, doneChan, time.Duration(common.STATS_TIMEOUT * time.Second)}
	go func() {
		if err := g.Docker.Stats(opt); err != nil {
			logs.Info("Get stats failed", self.ID[:12], err)
		}
	}()

	stats := &docker.Stats{}
	select {
	case stats = <-statsChan:
		if stats == nil {
			return false
		}
	case <-time.After(common.STATS_FORCE_DONE * time.Second):
		doneChan <- true
		return false
	}

	self.Info["cpu_user"] = stats.CPUStats.CPUUsage.UsageInUsermode
	self.Info["cpu_system"] = stats.CPUStats.CPUUsage.UsageInKernelmode
	self.Info["cpu_usage"] = stats.CPUStats.CPUUsage.TotalUsage
	//FIXME in container it will get all CPUStats
	self.Info["mem_usage"] = stats.MemoryStats.Usage
	self.Info["mem_max_usage"] = stats.MemoryStats.MaxUsage
	self.Info["mem_rss"] = stats.MemoryStats.Stats.Rss

	network, err := GetNetStats(self)
	if err != nil {
		logs.Info("Get net stats failed", self.ID[:12], err)
		return false
	}
	for k, d := range network {
		self.Info[k] = d
	}
	return true
}

func (self *EruApp) saveLast() {
	for k, d := range self.Info {
		self.Save[k] = d
	}
	self.Info = map[string]uint64{}
}

func (self *EruApp) calcRate(now time.Time) {
	delta := now.Sub(self.Last)
	nano_t := float64(delta.Nanoseconds())
	second_t := delta.Seconds()
	for k, d := range self.Info {
		switch {
		case strings.HasPrefix(k, "cpu_") && d >= self.Save[k]:
			self.Rate[fmt.Sprintf("%s_rate", k)] = float64(d-self.Save[k]) / nano_t
		case strings.HasPrefix(k, common.VLAN_PREFIX) && d >= self.Save[k]:
			self.Rate[fmt.Sprintf("%s.rate", k)] = float64(d-self.Save[k]) / second_t
		case strings.HasPrefix(k, "mem"):
			self.Rate[k] = float64(d)
		}
	}
	self.Last = now
	self.saveLast()
}

func (self *EruApp) send(rate map[string]float64) {
	data := []*model.MetricValue{}
	for k, d := range rate {
		data = append(data, self.newMetricValue(k, d))
	}
	var resp model.TransferResponse
	if err := self.Client.Call("Transfer.Update", data, &resp); err != nil {
		logs.Debug("Metrics call Transfer.Update fail", err, self.Name, self.EntryPoint)
		return
	}
	logs.Debug(data)
	logs.Debug(self.Endpoint, self.Last, &resp)
}

func (self *EruApp) newMetricValue(metric string, value interface{}) *model.MetricValue {
	mv := &model.MetricValue{
		Endpoint:  self.Endpoint,
		Metric:    metric,
		Value:     value,
		Step:      int64(self.Step.Seconds()),
		Type:      "GAUGE",
		Tags:      self.Tag,
		Timestamp: self.Last.Unix(),
	}
	return mv
}
