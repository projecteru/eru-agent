package metrics

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/CMGS/consistent"
	"github.com/docker/libcontainer"
	"github.com/open-falcon/agent/g"
	"github.com/open-falcon/common/model"

	"../defines"
	"../logs"
)

type MetricsRecorder struct {
	sync.RWMutex
	apps      map[string]*MetricData
	stop      chan bool
	step      int64
	hostname  string
	transfers *consistent.Consistent
	clients   map[string]g.SingleConnRpcClient
	factory   libcontainer.Factory
}

func NewMetricsRecorder(hostname string, config defines.MetricsConfig) *MetricsRecorder {
	r := &MetricsRecorder{}
	r.apps = map[string]*MetricData{}
	r.step = config.Step
	r.stop = make(chan bool)
	r.transfers = consistent.New()
	r.hostname = hostname
	r.clients = map[string]g.SingleConnRpcClient{}
	for _, transfer := range config.Transfers {
		r.transfers.Add(transfer)
		r.clients[transfer] = g.SingleConnRpcClient{
			RpcServer: transfer,
			Timeout:   time.Duration(config.Timeout) * time.Millisecond,
		}
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
	metric := NewMetricData(app, container)
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
	if _, ok := self.apps[ID]; !ok {
		return
	}
	delete(self.apps, ID)
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
		go func(ID string, metric *MetricData) {
			if !metric.UpdateStats() {
				logs.Info("Update Stats Failed", ID)
				return
			}
			metric.CalcRate()
			self.doSend(ID, metric)
			metric.SaveLast()
		}(ID, metric)
	}
}

func (self *MetricsRecorder) doSend(ID string, metric *MetricData) {
	tag := fmt.Sprintf(
		"hostname=%s,cid=%s,ident=%s",
		self.hostname, ID[:12], metric.app.Ident,
	)
	name := fmt.Sprintf("%s-%s", metric.app.Name, metric.app.EntryPoint)
	now := metric.last.Unix()
	for offset := 0; offset < self.transfers.Len(); offset++ {
		addr, err := self.transfers.Get(ID, offset)
		client := self.clients[addr]
		if err != nil {
			logs.Info("Get transfer failed", err, ID, metric.app.Name)
			break
		}
		m := []*model.MetricValue{}
		for k, d := range metric.info {
			if !strings.HasPrefix(k, "mem") {
				continue
			}
			m = append(m, self.newMetricValue(name, k, d, tag, now))
		}
		for k, d := range metric.rate {
			m = append(m, self.newMetricValue(name, k, d, tag, now))
		}
		var resp model.TransferResponse
		if err := client.Call("Transfer.Update", m, &resp); err != nil {
			logs.Debug("call Transfer.Update fail", err)
		} else {
			logs.Debug(name, &resp)
			break
		}
	}
}

func (self *MetricsRecorder) newMetricValue(endpoint, metric string, value interface{}, tags string, now int64) *model.MetricValue {
	mv := &model.MetricValue{
		Endpoint:  endpoint,
		Metric:    metric,
		Value:     value,
		Step:      self.step,
		Type:      "GAUGE",
		Tags:      tags,
		Timestamp: now,
	}
	return mv
}
