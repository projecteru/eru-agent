package app

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/HunanTV/eru-agent/utils"
	"github.com/fsouza/go-dockerclient"
)

type EruApp struct {
	defines.Meta
	defines.Metric
	statFile *os.File
}

func NewEruApp(container *docker.Container, extend map[string]interface{}) *EruApp {
	name, entrypoint, ident := utils.GetAppInfo(container.Name)
	if name == "" {
		logs.Info("Container name invaild", container.Name)
		return nil
	}
	logs.Debug("Eru App", name, entrypoint, ident)

	transfer, _ := g.Transfers.Get(container.ID, 0)
	client := defines.SingleConnRpcClient{
		RpcServer: transfer,
		Timeout:   time.Duration(g.Config.Metrics.Timeout) * time.Millisecond,
	}
	step := time.Duration(g.Config.Metrics.Step) * time.Second

	extend["hostname"] = g.Config.HostName
	extend["cid"] = container.ID[:12]
	extend["ident"] = ident
	tag := []string{}
	for k, v := range extend {
		tag = append(tag, fmt.Sprintf("%s=%v", k, v))
	}
	endpoint := fmt.Sprintf("%s-%s", name, entrypoint)

	eruApp := &EruApp{
		defines.Meta{container.ID, container.State.Pid, name, entrypoint, ident, extend},
		defines.Metric{Step: step, Client: client, Tag: strings.Join(tag, ","), Endpoint: endpoint},
		nil,
	}

	eruApp.Stop = make(chan bool)
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

func Valid(ID string) bool {
	lock.RLock()
	defer lock.RUnlock()
	_, ok := Apps[ID]
	return ok
}
