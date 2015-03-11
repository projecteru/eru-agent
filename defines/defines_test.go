package defines

import (
	"testing"

	"github.com/fsouza/go-dockerclient"
)

var config AgentConfig

func init() {
	config = AgentConfig{}
	config.Docker = DockerConfig{}
	config.Docker.Endpoint = "tcp://192.168.59.103:2375"
}

func Test_MockDocker(t *testing.T) {
	Docker := NewDocker(config.Docker.Endpoint)
	MockDocker(Docker)
	err := Docker.PushImage(docker.PushImageOptions{}, docker.AuthConfiguration{})
	if err != nil {
		t.Error(err)
	}
}
