package main

import (
	"fmt"
	"strings"

	"./common"
	"./defines"
	"./logs"
	"./utils"
	"github.com/fsouza/go-dockerclient"
	"github.com/keimoon/gore"
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

func (self *StatusMoniter) Watcher() {
	conn, err := common.Rds.Acquire()
	if err != nil || conn == nil {
		logs.Assert(err, "Get redis conn")
	}
	defer common.Rds.Release(conn)

	subs := gore.NewSubscriptions(conn)
	defer subs.Close()
	subKey := fmt.Sprintf("eru:agent:%s:watcher", config.HostName)
	logs.Debug("Monitor taget", subKey)
	subs.Subscribe(subKey)

	for message := range subs.Message() {
		if message == nil {
			break
		}
		command := string(message.Message)
		logs.Debug("Get command", command)
		parser := strings.Split(command, "|")
		control, containerID := parser[0], parser[1]
		switch control {
		case "+":
			logs.Info("Watch", containerID)
			container, err := common.Docker.InspectContainer(containerID)
			if err != nil {
				logs.Info("Status inspect docker failed", err)
			} else {
				self.Add(containerID, container.Name)
			}
		case "-":
			logs.Info("Remove", containerID)
			Metrics.Remove(containerID)
			if _, ok := self.Apps[containerID]; ok {
				delete(self.Apps, containerID)
			}
		}
	}
}

func (self *StatusMoniter) Load() {
	containers, err := common.Docker.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		logs.Assert(err, "List containers")
	}

	conn, err := common.Rds.Acquire()
	if err != nil || conn == nil {
		logs.Assert(err, "Get redis conn")
	}
	defer common.Rds.Release(conn)

	containersKey := fmt.Sprintf("eru:agent:%s:containers", config.HostName)
	logs.Debug("Get tagets from", containersKey)
	rep, err := gore.NewCommand("LRANGE", containersKey, 0, -1).Run(conn)
	if err != nil {
		logs.Assert(err, "Get targets")
	}
	targetContainersList := []string{}
	rep.Slice(&targetContainersList)
	logs.Debug("Targets:", targetContainersList)

	targets := map[string]struct{}{}
	for _, target := range targetContainersList {
		targets[target] = struct{}{}
	}

	logs.Info("Load container")

	for _, container := range containers {
		if _, ok := targets[container.ID]; !ok {
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
	if _, ok := self.Apps[ID]; ok {
		// safe add
		return
	}
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
