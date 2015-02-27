package main

import (
	"strings"

	"./common"
	"./defines"
	"./logs"
	"./utils"
	"github.com/fsouza/go-dockerclient"
)

type StatusMoniter struct {
	events chan *docker.APIEvents
	Apps   map[string]*defines.App
}

func NewStatus() *StatusMoniter {
	status := &StatusMoniter{}
	status.events = make(chan *docker.APIEvents)
	status.Apps = map[string]*defines.App{}
	logs.Assert(common.Docker.AddEventListener(status.events), "Attacher")
	return status
}

func (self *StatusMoniter) Listen() {
	logs.Info("Status Monitor Start")
	for event := range self.events {
		logs.Debug("Status:", event.Status, event.ID, event.From)
		if event.Status == common.STATUS_DIE {
			// Always in apps
			Metrics.Remove(event.ID)
			delete(self.Apps, event.ID)
		}
		if event.Status == common.STATUS_START {
			container, err := common.Docker.InspectContainer(event.ID)
			if err != nil {
				logs.Info("Status inspect docker failed", err)
				continue
			}
			self.Add(event.ID, container.Name)
		}
	}
}

func (self *StatusMoniter) getStatus(s string) string {
	switch {
	case strings.HasPrefix(s, "Up"):
		return common.STATUS_START
	default:
		return common.STATUS_DIE
	}
}

func (self *StatusMoniter) Load() {
	containers, err := common.Docker.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		logs.Info(err, "Load container")
	}

	logs.Info("Load container")
	for _, container := range containers {
		if !strings.HasPrefix(container.Image, config.Docker.Registry) {
			continue
		}
		status := self.getStatus(container.Status)
		if status != common.STATUS_START {
			//TODO report to eru
			continue
		}
		self.Add(container.ID, container.Names[0])
	}
}

func (self *StatusMoniter) Add(ID, containerName string) {
	name, entrypoint, ident := utils.GetAppInfo(containerName)
	if name == "" {
		// ignore
		return
	}
	logs.Debug("Container", name, entrypoint, ident)
	app := &defines.App{name, entrypoint, ident}
	self.Apps[ID] = app
	Metrics.Add(ID, app)
	Lenz.Attacher.Attach(ID, app)
}
