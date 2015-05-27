package metrics

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"../common"
	"../defines"
	"../logs"
	"github.com/docker/libcontainer"
	"github.com/docker/libcontainer/cgroups"
	"github.com/fsouza/go-dockerclient"
)

type MetricData struct {
	app       *defines.App
	last      time.Time
	exec      *docker.Exec
	container libcontainer.Container

	info map[string]uint64
	save map[string]uint64
	rate map[string]float64
}

func NewMetricData(app *defines.App, container libcontainer.Container) *MetricData {
	m := &MetricData{}
	m.app = app
	m.container = container
	m.info = map[string]uint64{}
	m.save = map[string]uint64{}
	m.rate = map[string]float64{}
	return m
}

func GetNetStats(exec *docker.Exec) (result map[string]uint64, err error) {
	outr, outw := io.Pipe()
	defer outr.Close()

	success := make(chan struct{})
	failure := make(chan error)
	go func() {
		// TODO: 防止被err流block, 删掉先, 之后记得补上
		err = common.Docker.StartExec(
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

func (self *MetricData) UpdateStats() bool {
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
	self.info["mem_usage"] = stats.MemoryStats.Usage
	self.info["mem_max_usage"] = stats.MemoryStats.MaxUsage
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

func (self *MetricData) SaveLast() {
	for k, d := range self.info {
		self.save[k] = d
	}
	self.info = map[string]uint64{}
}

func (self *MetricData) CalcRate() {
	delta := time.Now().Sub(self.last)
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
	self.UpdateTime()
}

func (self *MetricData) SetExec() (err error) {
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

func (self *MetricData) UpdateTime() {
	self.last = time.Now()
}
