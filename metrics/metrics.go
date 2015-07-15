package metrics

import (
	"sync"
	"time"

	"github.com/CMGS/consistent"
	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/g"
)

var Metrics *MetricsRecorder

type MetricsRecorder struct {
	sync.RWMutex
	apps       map[string]struct{}
	step       time.Duration
	rpcTimeout time.Duration
	transfers  *consistent.Consistent
}

func NewMetricsRecorder() *MetricsRecorder {
	r := &MetricsRecorder{}
	r.apps = map[string]struct{}{}
	r.transfers = consistent.New()
	r.step = time.Duration(g.Config.Metrics.Step) * time.Second
	r.rpcTimeout = time.Duration(g.Config.Metrics.Timeout) * time.Millisecond
	for _, transfer := range g.Config.Metrics.Transfers {
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

	metric := NewMetricData(app, client, self.step)
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
