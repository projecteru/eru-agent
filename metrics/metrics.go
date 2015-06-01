package metrics

import (
	"sync"
	"time"

	"github.com/CMGS/consistent"
	"github.com/docker/libcontainer"

	"../defines"
	"../logs"
)

type MetricsRecorder struct {
	sync.RWMutex
	apps       map[string]*MetricData
	stop       chan bool
	step       int64
	hostname   string
	rpcTimeout time.Duration
	transfers  *consistent.Consistent
	factory    libcontainer.Factory
}

func NewMetricsRecorder(hostname string, config defines.MetricsConfig) *MetricsRecorder {
	r := &MetricsRecorder{}
	r.apps = map[string]*MetricData{}
	r.step = config.Step
	r.stop = make(chan bool)
	r.transfers = consistent.New()
	r.hostname = hostname
	r.rpcTimeout = time.Duration(config.Timeout) * time.Millisecond
	for _, transfer := range config.Transfers {
		r.transfers.Add(transfer)
	}
	var err error
	if r.factory, err = libcontainer.New(config.Root); err != nil {
		logs.Assert(err, "Load containers dir failed")
	}
	return r
}

func (self *MetricsRecorder) Add(ID string, app *defines.App) {
	self.Lock()
	defer self.Unlock()
	if _, ok := self.apps[ID]; ok {
		return
	}

	container, err := self.factory.Load(ID)
	if err != nil {
		logs.Info("Load Container Failed", err)
		return
	}

	addr, err := self.transfers.Get(ID, 0)
	client := SingleConnRpcClient{
		RpcServer: addr,
		Timeout:   self.rpcTimeout,
	}

	metric := NewMetricData(app, container, client, self.step)
	if err := metric.SetExec(); err != nil {
		logs.Info("Create Exec Command Failed", err)
		return
	}
	metric.UpdateTime()
	if !metric.UpdateStats() {
		logs.Info("Update Stats Failed", ID)
		return
	}
	metric.SaveLast()
	self.apps[ID] = metric
}

func (self *MetricsRecorder) Remove(ID string) {
	self.Lock()
	defer self.Unlock()
	defer delete(self.apps, ID)
	defer self.apps[ID].Close()
	if _, ok := self.apps[ID]; !ok {
		return
	}
}

func (self *MetricsRecorder) Report() {
	defer close(self.stop)
	for {
		select {
		case <-time.After(time.Second * time.Duration(self.step)):
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
	for ID, metric := range self.apps {
		if !metric.UpdateStats() {
			logs.Info("Remove from metric list", ID)
			self.Remove(ID)
			continue
		}
		go func(ID string, metric *MetricData) {
			metric.CalcRate()
			metric.Send(self.hostname, ID)
			metric.SaveLast()
		}(ID, metric)
	}
}
