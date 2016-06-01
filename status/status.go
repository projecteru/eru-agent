package status

import (
	"encoding/json"
	"fmt"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	eventtypes "github.com/docker/engine-api/types/events"

	"github.com/keimoon/gore"
	"github.com/projecteru/eru-agent/app"
	"github.com/projecteru/eru-agent/common"
	"github.com/projecteru/eru-agent/g"
	"github.com/projecteru/eru-agent/lenz"
	"github.com/projecteru/eru-agent/utils"
)

var eventHandler = NewEventHandler()

func InitStatus() {
	eventHandler.Handle(common.STATUS_START, handleContainerStart)
	eventHandler.Handle(common.STATUS_DIE, handleContainerDie)
	log.Info("Status initiated")
}

func Start() {
	log.Info("Status monitor start")
	go monitor()
	load()
}

func handleContainerStart(event eventtypes.Message) {
	log.Debugf("Status start %s %s", event.ID[:12], event.From)
	ctx := context.Background()
	if meta := getContainerMeta(event.ID); meta != nil && !app.Valid(event.ID) {
		//TODO use global docker
		container, err := g.Docker.ContainerInspect(ctx, event.ID)
		if err != nil {
			log.Errorf("Status inspect docker failed %s", err)
			return
		}
		eruApp := app.NewEruApp(container, meta)
		if eruApp == nil {
			log.Info("Create EruApp failed")
			return
		}
		lenz.Attacher.Attach(&eruApp.Meta)
		app.Add(eruApp)
		reportContainerCure(event.ID)
	}
}

func handleContainerDie(event eventtypes.Message) {
	log.Debugf("Status die %s %s", event.ID[:12], event.From)
	app.Remove(event.ID)
	reportContainerDeath(event.ID)
}

func monitor() {
	var eventChan = make(chan eventtypes.Message)
	go eventHandler.Watch(eventChan)
	MonitorContainerEvents(g.ErrChan, eventChan)
	close(eventChan)
}

func getContainerMeta(cid string) map[string]interface{} {
	conn := g.GetRedisConn()
	defer g.ReleaseRedisConn(conn)

	containersKey := fmt.Sprintf("eru:agent:%s:containers:meta", g.Config.HostName)
	rep, err := gore.NewCommand("HGET", containersKey, cid).Run(conn)
	if err != nil {
		log.Errorf("Status get meta %s", err)
		return nil
	}
	var result map[string]interface{}
	if rep.IsNil() {
		return nil
	}
	if b, err := rep.Bytes(); err != nil {
		log.Errorf("Status get meta %s", err)
		return nil
	} else {
		if err := json.Unmarshal(b, &result); err != nil {
			log.Errorf("Status unmarshal meta %s", err)
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
		log.Errorf("Status failed in get flag %s", err)
		return
	}
	if !rep.IsNil() {
		gore.NewCommand("DEL", flagKey).Run(conn)
		log.Debugf("%s Status flag set, ignore", cid[:12])
		return
	}

	url := fmt.Sprintf("%s/api/container/%s/kill/", g.Config.Eru.Endpoint, cid)
	utils.DoPut(url)
	log.Debugf("%s dead, remove from watching list", cid[:12])
}

func reportContainerCure(cid string) {
	url := fmt.Sprintf("%s/api/container/%s/cure/", g.Config.Eru.Endpoint, cid)
	utils.DoPut(url)
	log.Debugf("%s, cured, added in watching list", cid[:12])
}
