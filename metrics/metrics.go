package metrics

import (
	"sync"
	"time"

	"github.com/CMGS/consistent"
	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
)

var step, rpcTimeout time.Duration
var transfers *consistent.Consistent
var report map[string]*MetricData
var lock sync.Mutex

func InitMetrics() {
	step = time.Duration(g.Config.Metrics.Step) * time.Second
	rpcTimeout = time.Duration(g.Config.Metrics.Timeout) * time.Millisecond
	transfers = consistent.New()
	report = map[string]*MetricData{}
	lock = sync.Mutex{}
	for _, transfer := range g.Config.Metrics.Transfers {
		transfers.Add(transfer)
	}
	logs.Info("Metrics initiated")
}

func Start(app *defines.App) {
	if _, ok := report[app.ID]; ok {
		return
	}
	lock.Lock()
	defer lock.Unlock()
	addr, _ := transfers.Get(app.ID, 0)
	client := SingleConnRpcClient{
		RpcServer: addr,
		Timeout:   rpcTimeout,
	}

	report[app.ID] = NewMetricData(app, client, step)
	go report[app.ID].Report()
}

func Stop(ID string) {
	if _, ok := report[ID]; !ok {
		return
	}
	lock.Lock()
	defer lock.Unlock()
	report[ID].Stop()
	delete(report, ID)
}
