package app

import (
	"fmt"
	"sync"
	"time"

	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/HunanTV/eru-agent/utils"
)

type EruApp struct {
	defines.Meta
	defines.Metric
}

func NewEruApp(ID, containerName string) *EruApp {
	name, entrypoint, ident := utils.GetAppInfo(containerName)
	if name == "" {
		logs.Info("Container name invald", containerName)
		return nil
	}
	logs.Debug("Eru App", name, entrypoint, ident)

	transfer, _ := g.Transfers.Get(ID, 0)
	client := defines.SingleConnRpcClient{
		RpcServer: transfer,
		Timeout:   time.Duration(g.Config.Metrics.Timeout) * time.Millisecond,
	}
	step := time.Duration(g.Config.Metrics.Step) * time.Second
	tag := fmt.Sprintf("hostname=%s,cid=%s,ident=%s", g.Config.HostName, ID[:12], ident)
	endpoint := fmt.Sprintf("%s-%s", name, entrypoint)

	eruApp := &EruApp{
		defines.Meta{ID, name, entrypoint, ident, ""},
		defines.Metric{Step: step, Client: client, Tag: tag, Endpoint: endpoint},
	}

	eruApp.Stop = make(chan bool)
	eruApp.Info = map[string]uint64{}
	eruApp.Save = map[string]uint64{}
	eruApp.Rate = map[string]float64{}

	return eruApp
}

var lock sync.RWMutex
var Apps map[string]*EruApp = map[string]*EruApp{}

func Add(app *EruApp) {
	lock.Lock()
	defer lock.Unlock()
	if _, ok := Apps[app.ID]; ok {
		// safe add
		return
	}
	if !app.InitMetric() {
		// not record
		return
	}
	go app.Report()
	Apps[app.ID] = app
}

func Remove(ID string) {
	lock.Lock()
	defer lock.Unlock()
	if _, ok := Apps[ID]; !ok {
		return
	}
	Apps[ID].Exit()
	delete(Apps, ID)
}

func Vaild(ID string) bool {
	lock.RLock()
	defer lock.RUnlock()
	_, ok := Apps[ID]
	return ok
}
