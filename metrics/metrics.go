package metrics

import (
	"sync"
	"time"

	"../defines"
	"../logs"
)

type MetricData struct {
	app *defines.App

	mem_usage uint64
	mem_rss   uint64

	net_inbytes  uint64
	net_outbytes uint64
	net_inerrs   uint64
	net_outerrs  uint64

	cpu_user   uint64
	cpu_system uint64
	cpu_usage  uint64
}

func NewMetricData(app *defines.App) *MetricData {
	m := &MetricData{}
	m.app = app
	return m
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

	iStats, err := GetNetStats(ID)
	if err != nil {
		logs.Info(err)
		return false
	}
	self.net_inbytes = iStats["inbytes.0"]
	self.net_outbytes = iStats["outbytes.0"]
	self.net_inerrs = iStats["inerrs.0"]
	self.net_outerrs = iStats["outerrs.0"]
	return true
}

type MetricsRecorder struct {
	mu     *sync.Mutex
	apps   map[string]*MetricData
	client *InfluxDBClient
	stop   chan bool
	t      int
	wg     *sync.WaitGroup
}

func NewMetricsRecorder(hostname string, config defines.MetricsConfig) *MetricsRecorder {
	InitDevDir()
	r := &MetricsRecorder{}
	r.mu = &sync.Mutex{}
	r.wg = &sync.WaitGroup{}
	r.apps = map[string]*MetricData{}
	r.client = NewInfluxDBClient(hostname, config)
	r.t = config.ReportInterval
	r.stop = make(chan bool)
	return r
}

func (self *MetricsRecorder) Add(ID string, app *defines.App) {
	self.mu.Lock()
	defer self.mu.Unlock()
	if _, ok := self.apps[ID]; ok {
		return
	}
	self.apps[ID] = NewMetricData(app)
	self.apps[ID].UpdateStats(ID)
}

func (self *MetricsRecorder) Remove(ID string) {
	self.mu.Lock()
	defer self.mu.Unlock()
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
	self.mu.Lock()
	defer self.mu.Unlock()
	self.wg.Add(len(self.apps))
	for ID, metric := range self.apps {
		go func(ID string, metric *MetricData) {
			defer self.wg.Done()
			self.client.GenSeries(ID, metric)
			metric.UpdateStats(ID)
		}(ID, metric)
	}
	self.wg.Wait()
	self.client.Send()
}
