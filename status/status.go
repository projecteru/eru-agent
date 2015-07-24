package status

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/HunanTV/eru-agent/app"
	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/lenz"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/fsouza/go-dockerclient"
	"github.com/keimoon/gore"
)

var events chan *docker.APIEvents = make(chan *docker.APIEvents)

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

	containersKey := fmt.Sprintf("eru:agent:%s:containers:meta", g.Config.HostName)
	logs.Debug("Status get targets from", containersKey)
	rep, err := gore.NewCommand("HGETALL", containersKey).Run(conn)
	if err != nil {
		logs.Assert(err, "Status get targets")
	}

	if rep.IsNil() {
		return
	}

	targets, err := rep.Map()
	if err != nil {
		logs.Assert(err, "Status load targets")
	}

	logs.Debug("Status targets:", targets)
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
		var meta map[string]interface{}
		if err := json.Unmarshal([]byte(targets[container.ID]), &meta); err != nil {
			logs.Info("Status load failed", err)
			continue
		}
		if eruApp := app.NewEruApp(container.ID, container.Names[0], meta); eruApp != nil {
			app.Add(eruApp)
			lenz.Attacher.Attach(&eruApp.Meta)
			reportContainerCure(container.ID)
		}
	}
}

func StartMonitor() {
	logs.Info("Status monitor start")
	go monitor()
}

func monitor() {
	for event := range events {
		logs.Debug("Status", event.Status, event.ID, event.From)
		switch event.Status {
		case common.STATUS_DIE:
			// Check if exists
			if app.Valid(event.ID) {
				app.Remove(event.ID)
				reportContainerDeath(event.ID)
			}
		case common.STATUS_START:
			// if not in watching list, just ignore it
			if meta := getContainerMeta(event.ID); meta != nil && !app.Valid(event.ID) {
				container, err := g.Docker.InspectContainer(event.ID)
				if err != nil {
					logs.Info("Status inspect docker failed", err)
					break
				}
				eruApp := app.NewEruApp(event.ID, container.Name, meta)
				if eruApp == nil {
					logs.Info("Create EruApp failed")
					break
				}
				app.Add(eruApp)
				lenz.Attacher.Attach(&eruApp.Meta)
				reportContainerCure(event.ID)
				logs.Debug(event.ID, "cured, added in watching list")
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

func getContainerMeta(cid string) map[string]interface{} {
	conn, err := g.Rds.Acquire()
	if err != nil || conn == nil {
		logs.Info("Status get redis conn", err)
		return nil
	}
	defer g.Rds.Release(conn)

	containersKey := fmt.Sprintf("eru:agent:%s:containers:meta", g.Config.HostName)
	rep, err := gore.NewCommand("HGET", containersKey, cid).Run(conn)
	if err != nil || rep.IsNil() {
		logs.Info("Status get target", err)
		return nil
	}
	var result map[string]interface{}
	if b, err := rep.Bytes(); err != nil {
		logs.Info("Status get meta", err)
	} else {
		if err := json.Unmarshal(b, &result); err != nil {
			logs.Info("Status get meta", err)
			return nil
		}
	}
	return result
}

func reportContainerDeath(cid string) {
	conn, err := g.Rds.Acquire()
	if err != nil || conn == nil {
		logs.Assert(err, "Get redis conn")
	}
	defer g.Rds.Release(conn)

	rep, err := gore.NewCommand("GET", fmt.Sprintf("eru:agent:%s:container:flag", cid)).Run(conn)
	if err != nil {
		logs.Info("Status failed in get flag", err)
		return
	}
	if !rep.IsNil() {
		logs.Debug(cid, "Status flag set, ignore")
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
