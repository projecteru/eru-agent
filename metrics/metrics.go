package metrics

import (
	"sync"
	"time"

	"github.com/CMGS/consistent"

	"../defines"
)

type MetricsRecorder struct {
	sync.RWMutex
	apps       map[string]struct{}
	step       time.Duration
	hostname   string
	rpcTimeout time.Duration
	transfers  *consistent.Consistent
}

func NewMetricsRecorder(hostname string, config defines.MetricsConfig) *MetricsRecorder {
	r := &MetricsRecorder{}
	r.hostname = hostname
	r.apps = map[string]struct{}{}
	r.transfers = consistent.New()
	r.step = time.Duration(config.Step) * time.Second
	r.rpcTimeout = time.Duration(config.Timeout) * time.Millisecond
	for _, transfer := range config.Transfers {
		r.transfers.Add(transfer)
	}
	return r
}

func (self *MetricsRecorder) Add(ID string, app *defines.App) {
	self.Lock()
	defer self.Unlock()
	if _, ok := self.apps[ID]; ok {
		return
	}

	addr, _ := self.transfers.Get(ID, 0)
	client := SingleConnRpcClient{
		RpcServer: addr,
		Timeout:   self.rpcTimeout,
	}

	metric := NewMetricData(app, client, self.step, self.hostname)
	go metric.Report()
	self.apps[ID] = struct{}{}
}

func (self *MetricsRecorder) Remove(ID string) {
	self.Lock()
	defer self.Unlock()
	defer delete(self.apps, ID)
	if _, ok := self.apps[ID]; !ok {
		return
	}
}

func (self *MetricsRecorder) Vaild(ID string) bool {
	self.RLock()
	defer self.RUnlock()
	_, ok := self.apps[ID]
	return ok
}

var Metrics *MetricsRecorder
