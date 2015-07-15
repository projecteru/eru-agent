package status

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/lenz"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/HunanTV/eru-agent/metrics"
	"github.com/fsouza/go-dockerclient"
	"github.com/keimoon/gore"
)

var events chan *docker.APIEvents = make(chan *docker.APIEvents)
var Apps map[string]*defines.App = map[string]*defines.App{}

func InitStatus() {
	logs.Assert(g.Docker.AddEventListener(events), "Attacher")
	logs.Info("Status initiated")
}

func Load() {
	containers, err := g.Docker.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		logs.Assert(err, "List containers")
	}

	conn, err := g.Rds.Acquire()
	if err != nil || conn == nil {
		logs.Assert(err, "Get redis conn")
	}
	defer g.Rds.Release(conn)

	containersKey := fmt.Sprintf("eru:agent:%s:containers", g.Config.HostName)
	logs.Debug("Status get targets from", containersKey)
	rep, err := gore.NewCommand("SMEMBERS", containersKey).Run(conn)
	if err != nil {
		logs.Assert(err, "Status get targets")
	}
	targetContainersList := []string{}
	rep.Slice(&targetContainersList)
	logs.Debug("Status targets:", targetContainersList)

	targets := map[string]struct{}{}
	for _, target := range targetContainersList {
		targets[target] = struct{}{}
	}

	logs.Info("Status load container")
	for _, container := range containers {
		if _, ok := targets[container.ID]; !ok {
			continue
		}
		status := getStatus(container.Status)
		if status != common.STATUS_START {
			reportContainerDeath(container.ID)
			continue
		}
		if app := defines.NewApp(container.ID, container.Names[0]); app != nil {
			Add(app)
			lenz.Attacher.Attach(app)
			metrics.Add(app)
			reportContainerCure(container.ID)
		}
	}
}

func Add(app *defines.App) {
	if _, ok := Apps[app.ID]; ok {
		// safe add
		return
	}
	Apps[app.ID] = app
}

func StartMonitor() {
	logs.Info("Status Monitor Start")
	go monitor()
}

func monitor() {
	for event := range events {
		logs.Debug("Status:", event.Status, event.ID, event.From)
		switch event.Status {
		case common.STATUS_DIE:
			// Check if exists
			if _, ok := Apps[event.ID]; ok {
				metrics.Remove(event.ID)
				delete(Apps, event.ID)
				reportContainerDeath(event.ID)
			}
		case common.STATUS_START:
			// if not in watching list, just ignore it
			if isInWatchingSet(event.ID) {
				container, err := g.Docker.InspectContainer(event.ID)
				if err != nil {
					logs.Info("Status inspect docker failed", err)
					break
				}
				if app := defines.NewApp(event.ID, container.Name); app != nil {
					Add(app)
					lenz.Attacher.Attach(app)
					metrics.Add(app)
					logs.Debug(event.ID, "cured, added in watching list")
				}
			}
		}
	}
}

func getStatus(s string) string {
	switch {
	case strings.HasPrefix(s, "Up"):
		return common.STATUS_START
	default:
		return common.STATUS_DIE
	}
}

func isInWatchingSet(cid string) bool {
	conn, err := g.Rds.Acquire()
	if err != nil || conn == nil {
		logs.Assert(err, "Get redis conn")
	}
	defer g.Rds.Release(conn)

	containersKey := fmt.Sprintf("eru:agent:%s:containers", g.Config.HostName)
	rep, err := gore.NewCommand("SISMEMBER", containersKey, cid).Run(conn)
	if err != nil {
		logs.Assert(err, "Get targets")
	}
	repInt, _ := rep.Int()
	return repInt == 1
}

func reportContainerDeath(cid string) {
	conn, err := g.Rds.Acquire()
	if err != nil || conn == nil {
		logs.Assert(err, "Get redis conn")
	}
	defer g.Rds.Release(conn)

	rep, err := gore.NewCommand("GET", fmt.Sprintf("eru:agent:%s:container:flag", cid)).Run(conn)
	if err != nil {
		logs.Assert(err, "failed in GET")
	}
	if !rep.IsNil() {
		logs.Debug(cid, "flag set, ignore")
		return
	}

	url := fmt.Sprintf("%s/api/container/%s/kill", g.Config.Eru.Endpoint, cid)
	client := &http.Client{}
	req, _ := http.NewRequest("PUT", url, nil)
	client.Do(req)
}

func reportContainerCure(cid string) {
	url := fmt.Sprintf("%s/api/container/%s/cure", g.Config.Eru.Endpoint, cid)
	client := &http.Client{}
	req, _ := http.NewRequest("PUT", url, nil)
	client.Do(req)
}
