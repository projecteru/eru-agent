package main

import (
	"fmt"
	"testing"

	"./common"
	"./defines"
	"github.com/fsouza/go-dockerclient"
)

func init() {
	InitTest()
}

func Test_GetStatus(t *testing.T) {
	a := "Exited (0) 9 days ago"
	if Status.getStatus(a) != common.STATUS_DIE {
		t.Error("Wrong Status")
	}
	a = "Up 8 days"
	if Status.getStatus(a) != common.STATUS_START {
		t.Error("Wrong Status")
	}
}

func Test_StatusReport(t *testing.T) {
	id := "xxxxxxxxxxxx"
	common.Docker.ListContainers = func(opt docker.ListContainersOptions) ([]docker.APIContainers, error) {
		c1 := docker.APIContainers{
			Names:  []string{"/test_1234"},
			ID:     id,
			Image:  config.Docker.Registry,
			Status: "Exited (0) 9 days ago",
		}
		c := []docker.APIContainers{c1}
		return c, nil
	}
	tid := "zzzzzzzzzzzz"
	common.Ws.WriteJSON = func(d interface{}) error {
		x, ok := d.(*defines.Result)
		if !ok {
			t.Error("Wrong Data")
		}
		if x.Id != tid {
			t.Error("Wrong Task ID")
		}
		if x.Done != true {
			t.Error("Wrong Done")
		}
		if x.Index != 0 {
			t.Error("Wrong Index")
		}
		if x.Type != common.INFO_TASK {
			t.Error("Wrong Task")
		}
		if x.Data != fmt.Sprintf("%s|test|%s", common.STATUS_DIE, id) {
			t.Error("Wrong Data")
		}
		return nil
	}
	Status.Report(tid)
	if _, ok := Status.Removable[id]; !ok {
		t.Error("Wrong Data")
	}
}

func Test_StatusDie(t *testing.T) {
	id := "xxx"
	common.Docker.InspectContainer = func(string) (*docker.Container, error) {
		return &docker.Container{Name: "/test_1234"}, nil
	}
	common.Ws.WriteJSON = func(d interface{}) error {
		x, ok := d.(*defines.Result)
		if !ok {
			t.Error("Wrong Data")
		}
		if x.Id != common.STATUS_IDENT {
			t.Error("Wrong Task ID")
		}
		if x.Index != 0 {
			t.Error("Wrong Index")
		}
		if x.Type != common.INFO_TASK {
			t.Error("Wrong Task")
		}
		if x.Data != fmt.Sprintf("%s|test|%s", common.STATUS_DIE, id) {
			t.Error("Wrong Data")
		}
		return nil
	}
	Status.die(id)
}

func Test_StatusListen(t *testing.T) {
	go Status.Listen()
	id := "abcdefghijklmnopqrstuvwxyz"
	event := &docker.APIEvents{"die", id, "test", 12345}
	common.Docker.InspectContainer = func(string) (*docker.Container, error) {
		t.Error("Wrong event")
		return nil, nil
	}
	Status.Removable[id] = struct{}{}
	common.Docker.InspectContainer = func(i string) (*docker.Container, error) {
		if i != id {
			t.Error("Wrong event")
		}
		return &docker.Container{ID: id, Name: "/test_1234"}, nil
	}
	common.Ws.WriteJSON = func(d interface{}) error {
		x, ok := d.(*defines.Result)
		if !ok {
			t.Error("Wrong Data")
		}
		if x.Id != common.STATUS_IDENT {
			t.Error("Wrong Task ID")
		}
		if x.Index != 0 {
			t.Error("Wrong Index")
		}
		if x.Type != common.INFO_TASK {
			t.Error("Wrong Task")
		}
		if x.Data != fmt.Sprintf("%s|test|%s", common.STATUS_DIE, id) {
			t.Error("Wrong Data")
		}
		return nil
	}
	Status.events <- event
}
