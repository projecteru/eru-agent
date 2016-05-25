package status

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/keimoon/gore"
	"github.com/projecteru/eru-agent/app"
	"github.com/projecteru/eru-agent/common"
	"github.com/projecteru/eru-agent/g"
	"github.com/projecteru/eru-agent/lenz"
	"golang.org/x/net/context"
)

func load() {
	ctx := context.Background()
	options := types.ContainerListOptions{All: true}
	containers, err := g.Docker.ContainerList(ctx, options)
	if err != nil {
		log.Panicf("Status get all containers failed %s", err)
	}

	conn := g.GetRedisConn()
	defer g.ReleaseRedisConn(conn)

	containersKey := fmt.Sprintf("eru:agent:%s:containers:meta", g.Config.HostName)
	log.Debugf("Status get targets from %s", containersKey)
	rep, err := gore.NewCommand("HGETALL", containersKey).Run(conn)
	if err != nil {
		log.Panicf("Status get targets failed %s", err)
	}
	if rep.IsNil() {
		return
	}

	targets, err := rep.Map()
	if err != nil {
		log.Panicf("Status load targets failed %s", err)
	}

	log.Debugf("Status targets: %s", targets)
	log.Info("Status load container")
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
			log.Errorf("Status load failed %s", err)
			continue
		}

		c, err := g.Docker.ContainerInspect(ctx, container.ID)
		if err != nil {
			log.Infof("Status inspect docker failed %s", err)
			continue
		}

		if eruApp := app.NewEruApp(c, meta); eruApp != nil {
			lenz.Attacher.Attach(&eruApp.Meta)
			app.Add(eruApp)
			reportContainerCure(container.ID)
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
