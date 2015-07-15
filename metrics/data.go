package metrics

import (
	"fmt"
	"strings"
	"time"

	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/fsouza/go-dockerclient"
	"github.com/open-falcon/common/model"
)

type MetricData struct {
	app       *defines.App
	last      time.Time
	exec      *docker.Exec
	step      time.Duration
	tag       string
	endpoint  string
	rpcClient SingleConnRpcClient

	info map[string]uint64
	save map[string]uint64
	rate map[string]float64
}

func NewMetricData(app *defines.App, client SingleConnRpcClient, step time.Duration) *MetricData {
	m := &MetricData{}
	m.app = app
	m.rpcClient = client
	m.step = step
	m.info = map[string]uint64{}
	m.save = map[string]uint64{}
	m.rate = map[string]float64{}
	m.tag = fmt.Sprintf(
		"hostname=%s,cid=%s,ident=%s",
		g.Config.HostName, app.ID[:12], app.Ident,
	)
	m.endpoint = fmt.Sprintf("%s-%s", app.Name, app.EntryPoint)
	return m
}

func (self *MetricData) setExec() (err error) {
	cid := self.app.ID
	self.exec, err = g.Docker.CreateExec(
		docker.CreateExecOptions{
			AttachStdout: true,
			Cmd: []string{
				"cat", "/proc/net/dev",
			},
			Container: cid,
		},
	)
	if err != nil {
		return
	}
	logs.Debug("Create exec id", self.exec.ID)
	return
}

func (self *MetricData) updateStats() bool {
	statsChan := make(chan *docker.Stats)
	opt := docker.StatsOptions{self.app.ID, statsChan, false}
	go func() {
		if err := g.Docker.Stats(opt); err != nil {
			logs.Info("Get Stats Failed", err)
		}
	}()
	stats := <-statsChan
	if stats == nil {
		return false
	}

	self.info["cpu_user"] = stats.CPUStats.CPUUsage.UsageInUsermode
	self.info["cpu_system"] = stats.CPUStats.CPUUsage.UsageInKernelmode
	self.info["cpu_usage"] = stats.CPUStats.CPUUsage.TotalUsage
	self.info["mem_usage"] = stats.MemoryStats.Usage
	self.info["mem_max_usage"] = stats.MemoryStats.MaxUsage
	self.info["mem_rss"] = stats.MemoryStats.Stats.Rss

	if network, err := GetNetStats(self.exec); err != nil {
		logs.Info(err)
		return false
	} else {
		for k, d := range network {
			self.info[k] = d
		}
	}
	return true
}

func (self *MetricData) saveLast() {
	for k, d := range self.info {
		self.save[k] = d
		delete(self.info, k)
	}
}

func (self *MetricData) calcRate(now time.Time) {
	delta := now.Sub(self.last)
	nano_t := float64(delta.Nanoseconds())
	if self.info["cpu_user"] > self.save["cpu_user"] {
		self.rate["cpu_user_rate"] = float64(self.info["cpu_user"]-self.save["cpu_user"]) / nano_t
	}
	if self.info["cpu_system"] > self.save["cpu_system"] {
		self.rate["cpu_system_rate"] = float64(self.info["cpu_system"]-self.save["cpu_system"]) / nano_t
	}
	if self.info["cpu_usage"] > self.save["cpu_usage"] {
		self.rate["cpu_usage_rate"] = float64(self.info["cpu_usage"]-self.save["cpu_usage"]) / nano_t
	}
	second_t := delta.Seconds()
	for k, d := range self.info {
		if !strings.HasPrefix(k, common.VLAN_PREFIX) || d < self.save[k] {
			continue
		}
		self.rate[k+".rate"] = float64(d-self.save[k]) / second_t
	}
}

func (self *MetricData) getData() []*model.MetricValue {
	data := []*model.MetricValue{}
	for k, d := range self.info {
		if !strings.HasPrefix(k, "mem") {
			continue
		}
		data = append(data, self.newMetricValue(k, d))
	}
	for k, d := range self.rate {
		data = append(data, self.newMetricValue(k, d))
	}
	return data
}

func (self *MetricData) send(data []*model.MetricValue) {
	var resp model.TransferResponse
	if err := self.rpcClient.Call("Transfer.Update", data, &resp); err != nil {
		logs.Debug("call Transfer.Update fail", err)
		return
	}
	logs.Debug(self.endpoint, self.last, &resp)
}

func (self *MetricData) newMetricValue(metric string, value interface{}) *model.MetricValue {
	mv := &model.MetricValue{
		Endpoint:  self.endpoint,
		Metric:    metric,
		Value:     value,
		Step:      int64(self.step.Seconds()),
		Type:      "GAUGE",
		Tags:      self.tag,
		Timestamp: self.last.Unix(),
	}
	return mv
}

func (self *MetricData) Report() {
	defer self.close()
	logs.Info(self.app.Name, "Metrics report start")

	self.last = time.Now()
	self.setExec()
	if !self.updateStats() {
		return
	}
	self.saveLast()
	for {
		select {
		case now := <-time.After(self.step):
			if !vaild(self.app.ID) {
				return
			}
			if !self.updateStats() {
				// veth missing problem
				continue
			}
			self.calcRate(now)
			self.last = now
			data := self.getData()
			go self.send(data)
			self.saveLast()
		}
	}
	logs.Info(self.app.Name, self.app.EntryPoint, "Metrics report stop")
}

func (self *MetricData) close() {
	self.rpcClient.close()
}
