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
	info, upOk := self.updateStats()
	if !upOk {
		logs.Info("Init mertics failed", self.Meta.ID[:12])
		return false
	}
	self.Last = time.Now()
	self.saveLast(info)
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
				info, upOk := self.updateStats()
				if isLimit {
					limitChan <- SoftLimit{upOk, self.ID, info}
				}
				if !upOk {
					logs.Info("Update mertic failed", self.Meta.ID[:12])
					return
				}
				rate := self.calcRate(info, now)
				self.saveLast(info)
				// for safe
				go self.send(rate)
			}()
		case <-self.Stop:
			return
		}
	}
}

func (self *EruApp) updateStats() (map[string]uint64, bool) {
	info := map[string]uint64{}
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
			return info, false
		}
	case <-time.After(common.STATS_FORCE_DONE * time.Second):
		doneChan <- true
		return info, false
	}

	info["cpu_user"] = stats.CPUStats.CPUUsage.UsageInUsermode
	info["cpu_system"] = stats.CPUStats.CPUUsage.UsageInKernelmode
	info["cpu_usage"] = stats.CPUStats.CPUUsage.TotalUsage
	//FIXME in container it will get all CPUStats
	info["mem_usage"] = stats.MemoryStats.Usage
	info["mem_max_usage"] = stats.MemoryStats.MaxUsage
	info["mem_rss"] = stats.MemoryStats.Stats.Rss

	if err := GetNetStats(self.Meta.Pid, info); err != nil {
		logs.Info("Get net stats failed", self.ID[:12], err)
		return info, false
	}
	return info, true
}

func (self *EruApp) saveLast(info map[string]uint64) {
	self.Save = map[string]uint64{}
	for k, d := range info {
		self.Save[k] = d
	}
}

func (self *EruApp) calcRate(info map[string]uint64, now time.Time) (rate map[string]float64) {
	rate = map[string]float64{}
	delta := now.Sub(self.Last)
	nano_t := float64(delta.Nanoseconds())
	second_t := delta.Seconds()
	for k, d := range info {
		switch {
		case strings.HasPrefix(k, "cpu_") && d >= self.Save[k]:
			rate[fmt.Sprintf("%s_rate", k)] = float64(d-self.Save[k]) / nano_t
		case strings.HasPrefix(k, common.VLAN_PREFIX) && d >= self.Save[k]:
			rate[fmt.Sprintf("%s.rate", k)] = float64(d-self.Save[k]) / second_t
		case strings.HasPrefix(k, "mem"):
			rate[k] = float64(d)
		}
	}
	self.Last = now
	return
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
