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
	step       time.Duration
	hostname   string
	rpcTimeout time.Duration
	transfers  *consistent.Consistent
	factory    libcontainer.Factory
}

func NewMetricsRecorder(hostname string, config defines.MetricsConfig) *MetricsRecorder {
	r := &MetricsRecorder{}
	r.hostname = hostname
	r.apps = map[string]*MetricData{}
	r.transfers = consistent.New()
	r.step = time.Duration(config.Step) * time.Second
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

	metric := NewMetricData(ID, app, container, client, self.step, self.hostname)
	self.apps[ID] = metric
	go metric.Report()
}

func (self *MetricsRecorder) Remove(ID string) {
	self.Lock()
	defer self.Unlock()
	defer delete(self.apps, ID)
	if _, ok := self.apps[ID]; !ok {
		return
	}
}
