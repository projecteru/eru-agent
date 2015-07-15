package metrics

import (
	"sync"
	"time"

	"github.com/CMGS/consistent"
	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
)

var lock sync.RWMutex
var step, rpcTimeout time.Duration
var transfers *consistent.Consistent

var Apps map[string]struct{}

func InitMetrics() {
	lock = sync.RWMutex{}
	step = time.Duration(g.Config.Metrics.Step) * time.Second
	rpcTimeout = time.Duration(g.Config.Metrics.Timeout) * time.Millisecond
	transfers = consistent.New()
	for _, transfer := range g.Config.Metrics.Transfers {
		transfers.Add(transfer)
	}

	Apps = map[string]struct{}{}
	logs.Info("Metrics initiated")
}

func Add(app *defines.App) {
	lock.Lock()
	defer lock.Unlock()
	if _, ok := Apps[app.ID]; ok {
		return
	}

	addr, _ := transfers.Get(app.ID, 0)
	client := SingleConnRpcClient{
		RpcServer: addr,
		Timeout:   rpcTimeout,
	}

	metric := NewMetricData(app, client, step)
	go metric.Report()
	Apps[app.ID] = struct{}{}
}

func Remove(ID string) {
	lock.Lock()
	defer lock.Unlock()
	if _, ok := Apps[ID]; !ok {
		return
	}
	delete(Apps, ID)
}

func vaild(ID string) bool {
	lock.RLock()
	defer lock.RUnlock()
	_, ok := Apps[ID]
	return ok
}
