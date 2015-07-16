package metrics

import (
	"time"

	"github.com/CMGS/consistent"
	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
)

var step, rpcTimeout time.Duration
var transfers *consistent.Consistent

func InitMetrics() {
	step = time.Duration(g.Config.Metrics.Step) * time.Second
	rpcTimeout = time.Duration(g.Config.Metrics.Timeout) * time.Millisecond
	transfers = consistent.New()
	for _, transfer := range g.Config.Metrics.Transfers {
		transfers.Add(transfer)
	}
	logs.Info("Metrics initiated")
}

func Start(app *defines.App) {
	addr, _ := transfers.Get(app.ID, 0)
	client := SingleConnRpcClient{
		RpcServer: addr,
		Timeout:   rpcTimeout,
	}

	metric := NewMetricData(app, client, step)
	go metric.Report()
}
