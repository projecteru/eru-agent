package status

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/projecteru/eru-agent/app"
	"github.com/projecteru/eru-agent/common"
	"github.com/projecteru/eru-agent/g"
	"github.com/projecteru/eru-agent/lenz"
	"github.com/projecteru/eru-agent/logs"
	"github.com/projecteru/eru-agent/utils"
	"github.com/fsouza/go-dockerclient"
	"github.com/keimoon/gore"
)

var events chan *docker.APIEvents = make(chan *docker.APIEvents)

func InitStatus() {
	logs.Assert(g.Docker.AddEventListener(events), "Attacher")
	logs.Info("Status initiated")
}

func load() {
	containers, err := g.Docker.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		logs.Assert(err, "List containers")
	}

	conn := g.GetRedisConn()
	defer g.ReleaseRedisConn(conn)
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

		c, err := g.Docker.InspectContainer(container.ID)
		if err != nil {
			logs.Info("Status inspect docker failed", err)
			continue
		}

		if eruApp := app.NewEruApp(c, meta); eruApp != nil {
			lenz.Attacher.Attach(&eruApp.Meta)
			app.Add(eruApp)
			reportContainerCure(container.ID)
		}
	}
}

func Start() {
	logs.Info("Status monitor start")
	go monitor()
	load()
}

func monitor() {
	for event := range events {
		switch event.Status {
		case common.STATUS_DIE:
			logs.Debug("Status", event.Status, event.ID[:12], event.From)
			app.Remove(event.ID)
			reportContainerDeath(event.ID)
		case common.STATUS_START:
			logs.Debug("Status", event.Status, event.ID[:12], event.From)
			// if not in watching list, just ignore it
			if meta := getContainerMeta(event.ID); meta != nil && !app.Valid(event.ID) {
				container, err := g.Docker.InspectContainer(event.ID)
				if err != nil {
					logs.Info("Status inspect docker failed", err)
					break
				}
				eruApp := app.NewEruApp(container, meta)
				if eruApp == nil {
					logs.Info("Create EruApp failed")
					break
				}
				lenz.Attacher.Attach(&eruApp.Meta)
				app.Add(eruApp)
				reportContainerCure(event.ID)
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
	conn := g.GetRedisConn()
	defer g.ReleaseRedisConn(conn)

	containersKey := fmt.Sprintf("eru:agent:%s:containers:meta", g.Config.HostName)
	rep, err := gore.NewCommand("HGET", containersKey, cid).Run(conn)
	if err != nil {
		logs.Info("Status get meta", err)
		return nil
	}
	var result map[string]interface{}
	if rep.IsNil() {
		return nil
	}
	if b, err := rep.Bytes(); err != nil {
		logs.Info("Status get meta", err)
		return nil
	} else {
		if err := json.Unmarshal(b, &result); err != nil {
			logs.Info("Status unmarshal meta", err)
			return nil
		}
	}
	return result
}

func reportContainerDeath(cid string) {
	conn := g.GetRedisConn()
	defer g.ReleaseRedisConn(conn)

	flagKey := fmt.Sprintf("eru:agent:%s:container:flag", cid)
	rep, err := gore.NewCommand("GET", flagKey).Run(conn)
	if err != nil {
		logs.Info("Status failed in get flag", err)
		return
	}
	if !rep.IsNil() {
		gore.NewCommand("DEL", flagKey).Run(conn)
		logs.Debug(cid[:12], "Status flag set, ignore")
		return
	}

	url := fmt.Sprintf("%s/api/container/%s/kill/", g.Config.Eru.Endpoint, cid)
	utils.DoPut(url)
	logs.Debug(cid[:12], "dead, remove from watching list")
}

func reportContainerCure(cid string) {
	url := fmt.Sprintf("%s/api/container/%s/cure/", g.Config.Eru.Endpoint, cid)
	utils.DoPut(url)
	logs.Debug(cid[:12], "cured, added in watching list")
}
