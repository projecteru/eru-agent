package metrics

import (
	"sync"
	"time"

	"github.com/fsouza/go-dockerclient"

	"../common"
	"../defines"
	"../logs"
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

	t    time.Time
	exec *docker.Exec
}

func NewMetricData(app *defines.App) *MetricData {
	m := &MetricData{}
	m.app = app
	return m
}

func (self *MetricData) SetExec(cid string) (err error) {
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

func (self *MetricData) UpdateStats(ID string) bool {
	stats, err := GetCgroupStats(ID)
	if err != nil {
		logs.Info("Get Stats Failed", err)
		return false
	}
	self.cpu_user = stats.CpuStats.CpuUsage.UsageInUsermode
	self.cpu_system = stats.CpuStats.CpuUsage.UsageInKernelmode
	self.cpu_usage = stats.CpuStats.CpuUsage.TotalUsage

	self.mem_usage = stats.MemoryStats.Usage
	self.mem_max_usage = stats.MemoryStats.MaxUsage
	self.mem_rss = stats.MemoryStats.Stats["rss"]

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

func (self *MetricData) UpdateTime() {
	self.t = time.Now()
}

type MetricsRecorder struct {
	sync.RWMutex
	apps   map[string]*MetricData
	client *InfluxDBClient
	stop   chan bool
	t      int
	wg     *sync.WaitGroup
}

func NewMetricsRecorder(hostname string, config defines.MetricsConfig) *MetricsRecorder {
	InitDevDir()
	r := &MetricsRecorder{}
	r.wg = &sync.WaitGroup{}
	r.apps = map[string]*MetricData{}
	r.client = NewInfluxDBClient(hostname, config)
	r.t = config.ReportInterval
	r.stop = make(chan bool)
	return r
}

func (self *MetricsRecorder) Add(ID string, app *defines.App) {
	self.Lock()
	defer self.Unlock()
	if _, ok := self.apps[ID]; ok {
		return
	}
	//TODO workaround for waiting device ready
	metric := NewMetricData(app)
	time.Sleep(1 * time.Second)
	if err := metric.SetExec(ID); err != nil {
		logs.Info("Create Exec Command Failed", err)
		return
	}
	metric.UpdateTime()
	if !metric.UpdateStats(ID) {
		return
	}
	metric.SaveLast()
	self.apps[ID] = metric
}

func (self *MetricsRecorder) Remove(ID string) {
	self.Lock()
	defer self.Unlock()
	if _, ok := self.apps[ID]; !ok {
		return
	}
	delete(self.apps, ID)
}

func (self *MetricsRecorder) Report() {
	defer close(self.stop)
	for {
		select {
		case <-time.After(time.Second * time.Duration(self.t)):
			self.Send()
		case <-self.stop:
			logs.Info("Metrics Stop")
			return
		}
	}
}

func (self *MetricsRecorder) Stop() {
	self.stop <- true
}

func (self *MetricsRecorder) Send() {
	self.RLock()
	defer self.RUnlock()
	apps := len(self.apps)
	if apps <= 0 {
		return
	}
	self.wg.Add(apps)
	for ID, metric := range self.apps {
		go func(ID string, metric *MetricData) {
			defer self.wg.Done()
			// use RWMutex
			metric.UpdateStats(ID)
			metric.CalcRate()
			self.client.GenSeries(ID, metric)
			metric.SaveLast()
		}(ID, metric)
	}
	self.wg.Wait()
	self.client.Send()
}
