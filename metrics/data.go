package metrics

import (
	"fmt"
	"strings"
	"time"

	"../common"
	"../defines"
	"../logs"
	"github.com/docker/libcontainer"
	"github.com/docker/libcontainer/cgroups"
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
	container libcontainer.Container
	rpcClient SingleConnRpcClient

	info map[string]uint64
	save map[string]uint64
	rate map[string]float64
}

func NewMetricData(ID string, app *defines.App, container libcontainer.Container, client SingleConnRpcClient, step time.Duration, hostname string) *MetricData {
	m := &MetricData{}
	m.app = app
	m.container = container
	m.rpcClient = client
	m.step = step
	m.info = map[string]uint64{}
	m.save = map[string]uint64{}
	m.rate = map[string]float64{}
	m.tag = fmt.Sprintf(
		"hostname=%s,cid=%s,ident=%s",
		hostname, ID[:12], app.Ident,
	)
	m.endpoint = fmt.Sprintf("%s-%s", app.Name, app.EntryPoint)
	return m
}

func (self *MetricData) setExec() (err error) {
	cid := self.container.ID()
	self.exec, err = common.Docker.CreateExec(
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
	var stats *cgroups.Stats
	if s, err := self.container.Stats(); err != nil {
		logs.Info("Get Stats Failed", err)
		return false
	} else {
		stats = s.CgroupStats
	}

	self.info["cpu_user"] = stats.CpuStats.CpuUsage.UsageInUsermode
	self.info["cpu_system"] = stats.CpuStats.CpuUsage.UsageInKernelmode
	self.info["cpu_usage"] = stats.CpuStats.CpuUsage.TotalUsage
	self.info["mem_usage"] = stats.MemoryStats.Usage.Usage
	self.info["mem_max_usage"] = stats.MemoryStats.Usage.MaxUsage
	self.info["mem_rss"] = stats.MemoryStats.Stats["rss"]

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

func (self *MetricData) send() {
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
	var resp model.TransferResponse
	if err := self.rpcClient.Call("Transfer.Update", data, &resp); err != nil {
		logs.Debug("call Transfer.Update fail", err)
	} else {
		logs.Debug(self.endpoint, self.last, &resp)
	}
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
	defer logs.Info(self.app.Name, "Metrics report stop")
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
			if !self.updateStats() {
				// get stats failed will close report
				return
			}
			self.calcRate(now)
			self.last = now
			self.send()
			self.saveLast()
		}
	}
}

func (self *MetricData) close() {
	self.rpcClient.close()
}
