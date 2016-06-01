package app

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/projecteru/eru-agent/defines"
	"github.com/projecteru/eru-agent/g"
	"github.com/projecteru/eru-agent/utils"
	"github.com/projecteru/eru-metric/metric"
	"github.com/projecteru/eru-metric/statsd"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
)

type EruApp struct {
	defines.Meta
	metric.Metric
	sync.Mutex
}

func NewEruApp(container types.ContainerJSON, extend map[string]interface{}) *EruApp {
	name, entrypoint, ident := utils.GetAppInfo(container.Name)
	if name == "" {
		log.Infof("Container name invaild %s", container.Name)
		return nil
	}
	log.Debugf("Eru App %s %s %s", name, entrypoint, ident)

	transfer := g.Transfers.Get(container.ID, 0)
	client := statsd.CreateStatsDClient(transfer)

	step := time.Duration(g.Config.Metrics.Step) * time.Second
	//TODO remove version meta data
	version := extend["__version__"]
	delete(extend, "__version__")
	var tagString string
	if len(extend) > 0 {
		tag := []string{}
		for _, v := range extend {
			tag = append(tag, fmt.Sprintf("%v", v))
		}
		tagString = fmt.Sprintf("%s.%s.%s", g.Config.HostName, strings.Join(tag, "."), container.ID[:12])
	} else {
		tagString = fmt.Sprintf("%s.%s", g.Config.HostName, container.ID[:12])
	}
	endpoint := fmt.Sprintf("%s.%s.%s", name, version, entrypoint)

	meta := defines.Meta{container.ID, container.State.Pid, name, entrypoint, ident, extend}
	metric := metric.CreateMetric(step, client, tagString, endpoint)
	eruApp := &EruApp{meta, metric, sync.Mutex{}}
	return eruApp
}

var lock sync.RWMutex = sync.RWMutex{}
var Apps map[string]*EruApp = map[string]*EruApp{}

func Add(app *EruApp) {
	lock.Lock()
	defer lock.Unlock()
	if _, ok := Apps[app.ID]; ok {
		// safe add
		return
	}
	if err := app.InitMetric(app.ID, app.Pid); err != nil {
		log.Errorf("Init app metric failed %s", err)
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
	app := Apps[ID]
	app.Lock()
	defer app.Unlock()
	app.Exit()
	delete(Apps, ID)
}

func Valid(ID string) bool {
	lock.RLock()
	defer lock.RUnlock()
	_, ok := Apps[ID]
	return ok
}
