package status

import (
	"testing"

	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/g"
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

func Test_StatusListen(t *testing.T) {
	go Status.Listen()
	id := "abcdefghijklmnopqrstuvwxyz"
	event := &docker.APIEvents{"die", id, "test", 12345}
	g.Docker.InspectContainer = func(string) (*docker.Container, error) {
		t.Error("Wrong event")
		return nil, nil
	}
	Status.Apps[id] = &defines.App{}
	g.Docker.InspectContainer = func(i string) (*docker.Container, error) {
		if i != id {
			t.Error("Wrong event")
		}
		return &docker.Container{ID: id, Name: "/test_1234"}, nil
	}
	Status.events <- event
}
