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
	app *defines.App

	mem_usage     uint64
	mem_max_usage uint64
	mem_rss       uint64

	cpu_user   uint64
	cpu_system uint64
	cpu_usage  uint64

	last_cpu_user   uint64
	last_cpu_system uint64
	last_cpu_usage  uint64

	cpu_user_rate   float64
	cpu_system_rate float64
	cpu_usage_rate  float64

	network      map[string]uint64
	last_network map[string]uint64
	network_rate map[string]float64

	t         time.Time
	exec      *docker.Exec
	container libcontainer.Container
}

func NewMetricData(app *defines.App, container libcontainer.Container) *MetricData {
	m := &MetricData{}
	m.app = app
	m.container = container
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

	self.cpu_user = stats.CpuStats.CpuUsage.UsageInUsermode
	self.cpu_system = stats.CpuStats.CpuUsage.UsageInKernelmode
	self.cpu_usage = stats.CpuStats.CpuUsage.TotalUsage

	self.mem_usage = stats.MemoryStats.Usage
	self.mem_max_usage = stats.MemoryStats.MaxUsage
	self.mem_rss = stats.MemoryStats.Stats["rss"]

	var err error
	if self.network, err = GetNetStats(self.exec); err != nil {
		logs.Info(err)
		return false
	}
	return true
}

func (self *MetricData) SaveLast() {
	self.last_cpu_user = self.cpu_user
	self.last_cpu_system = self.cpu_system
	self.last_cpu_usage = self.cpu_usage
	self.last_network = map[string]uint64{}
	for key, data := range self.network {
		self.last_network[key] = data
	}
}

func (self *MetricData) CalcRate() {
	t := time.Now().Sub(self.t)
	nano_t := float64(t.Nanoseconds())
	if self.cpu_user > self.last_cpu_user {
		self.cpu_user_rate = float64(self.cpu_user-self.last_cpu_user) / nano_t
	}
	if self.cpu_system > self.last_cpu_system {
		self.cpu_system_rate = float64(self.cpu_system-self.last_cpu_system) / nano_t
	}
	if self.cpu_usage > self.last_cpu_usage {
		self.cpu_usage_rate = float64(self.cpu_usage-self.last_cpu_usage) / nano_t
	}
	second_t := t.Seconds()
	self.network_rate = map[string]float64{}
	for key, data := range self.network {
		if data >= self.last_network[key] {
			self.network_rate[key+".rate"] = float64(data-self.last_network[key]) / second_t
		}
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
	self.t = time.Now()
}
