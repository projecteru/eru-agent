package app

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/fsouza/go-dockerclient"
	"github.com/open-falcon/common/model"
)

func (self *EruApp) InitMetric() bool {
	var err error
	if self.Exec, err = g.Docker.CreateExec(
		docker.CreateExecOptions{
			AttachStdout: true,
			Cmd: []string{
				"cat", "/proc/net/dev",
			},
			Container: self.ID,
		},
	); err != nil {
		logs.Info("Create exec failed", err)
		return false
	}
	logs.Debug("Create exec id", self.Exec.ID[:12])
	if !self.updateStats() {
		return false
	}
	self.Last = time.Now()
	self.saveLast()
	return true
}

func (self *EruApp) Exit() {
	self.Stop <- true
}

func (self *EruApp) Report() {
	defer self.Client.Close()
	defer logs.Info(self.Name, self.EntryPoint, "metrics report stop")
	logs.Info(self.Name, self.EntryPoint, "metrics report start")
	for {
		select {
		case now := <-time.Tick(self.Step):
			go func() {
				upOk := self.updateStats()
				limitChan <- SoftLimit{upOk, self.ID, self.Info}
				if !upOk {
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
	opt := docker.StatsOptions{self.ID, statsChan, false, nil, time.Duration(2 * time.Second)}
	go func() {
		if err := g.Docker.Stats(opt); err != nil {
			logs.Info("Get Stats Failed", err)
		}
	}()
	stats := <-statsChan
	if stats == nil {
		return false
	}

	self.Info["cpu_user"] = stats.CPUStats.CPUUsage.UsageInUsermode
	self.Info["cpu_system"] = stats.CPUStats.CPUUsage.UsageInKernelmode
	self.Info["cpu_usage"] = stats.CPUStats.CPUUsage.TotalUsage
	//FIXME in container it will get all CPUStats
	//	for seq, d := range stats.CPUStats.CPUUsage.PercpuUsage {
	//		self.Info[fmt.Sprintf("cpu_%d", seq)] = d
	//	}
	self.Info["mem_usage"] = stats.MemoryStats.Usage
	self.Info["mem_max_usage"] = stats.MemoryStats.MaxUsage
	self.Info["mem_rss"] = stats.MemoryStats.Stats.Rss

	network, err := GetNetStats(self.Exec)
	if err != nil {
		logs.Info(err)
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
		case strings.HasPrefix(k, "cpu_") && d > self.Save[k]:
			self.Rate[fmt.Sprintf("%s_rate", k)] = float64(d-self.Save[k]) / nano_t
		case strings.HasPrefix(k, common.VLAN_PREFIX) && d > self.Save[k]:
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

func GetNetStats(exec *docker.Exec) (result map[string]uint64, err error) {
	outr, outw := io.Pipe()
	defer outr.Close()

	success := make(chan struct{})
	failure := make(chan error)
	go func() {
		// TODO: 防止被err流block, 删掉先, 之后记得补上
		err = g.Docker.StartExec(
			exec.ID,
			docker.StartExecOptions{
				OutputStream: outw,
				Success:      success,
			},
		)
		outw.Close()
		if err != nil {
			close(success)
			failure <- err
		}
	}()
	if _, ok := <-success; ok {
		success <- struct{}{}
		result = map[string]uint64{}
		s := bufio.NewScanner(outr)
		var d uint64
		for s.Scan() {
			var name string
			var n [8]uint64
			text := s.Text()
			if strings.Index(text, ":") < 1 {
				continue
			}
			ts := strings.Split(text, ":")
			fmt.Sscanf(ts[0], "%s", &name)
			if !strings.HasPrefix(name, common.VLAN_PREFIX) {
				continue
			}
			fmt.Sscanf(ts[1],
				"%d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d",
				&n[0], &n[1], &n[2], &n[3], &d, &d, &d, &d,
				&n[4], &n[5], &n[6], &n[7], &d, &d, &d, &d,
			)
			result[name+".inbytes"] = n[0]
			result[name+".inpackets"] = n[1]
			result[name+".inerrs"] = n[2]
			result[name+".indrop"] = n[3]
			result[name+".outbytes"] = n[4]
			result[name+".outpackets"] = n[5]
			result[name+".outerrs"] = n[6]
			result[name+".outdrop"] = n[7]
		}
		logs.Debug("Container net status", result)
		return
	}
	err = <-failure
	return nil, err
}
