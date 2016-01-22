package app

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/projecteru/eru-agent/defines"
	"github.com/projecteru/eru-agent/g"
	"github.com/projecteru/eru-agent/logs"
	"github.com/projecteru/eru-agent/utils"
	"github.com/projecteru/eru-metric/falcon"
	"github.com/projecteru/eru-metric/metric"
	"github.com/fsouza/go-dockerclient"
)

type EruApp struct {
	defines.Meta
	metric.Metric
}

func NewEruApp(container *docker.Container, extend map[string]interface{}) *EruApp {
	name, entrypoint, ident := utils.GetAppInfo(container.Name)
	if name == "" {
		logs.Info("Container name invaild", container.Name)
		return nil
	}
	logs.Debug("Eru App", name, entrypoint, ident)

	transfer, _ := g.Transfers.Get(container.ID, 0)
	client := falcon.CreateFalconClient(
		transfer,
		time.Duration(g.Config.Metrics.Timeout)*time.Millisecond,
	)

	step := time.Duration(g.Config.Metrics.Step) * time.Second
	extend["hostname"] = g.Config.HostName
	extend["cid"] = container.ID[:12]
	extend["ident"] = ident
	tag := []string{}
	for k, v := range extend {
		tag = append(tag, fmt.Sprintf("%s=%v", k, v))
	}
	endpoint := fmt.Sprintf("%s-%s", name, entrypoint)

	meta := defines.Meta{container.ID, container.State.Pid, name, entrypoint, ident, extend}
	metric := metric.CreateMetric(step, client, strings.Join(tag, ","), endpoint)
	eruApp := &EruApp{meta, metric}
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
	if err := app.InitMetric(app.ID, app.Pid); err != nil {
		logs.Info("Init app metric failed", err)
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
