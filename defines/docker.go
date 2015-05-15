package defines

import (
	"../logs"
	"github.com/fsouza/go-dockerclient"
)

type DockerWrapper struct {
	*docker.Client
	PushImage        func(docker.PushImageOptions, docker.AuthConfiguration) error
	PullImage        func(docker.PullImageOptions, docker.AuthConfiguration) error
	CreateContainer  func(docker.CreateContainerOptions) (*docker.Container, error)
	StartContainer   func(string, *docker.HostConfig) error
	BuildImage       func(docker.BuildImageOptions) error
	KillContainer    func(docker.KillContainerOptions) error
	StopContainer    func(string, uint) error
	InspectContainer func(string) (*docker.Container, error)
	ListContainers   func(docker.ListContainersOptions) ([]docker.APIContainers, error)
	ListImages       func(docker.ListImagesOptions) ([]docker.APIImages, error)
	RemoveContainer  func(docker.RemoveContainerOptions) error
	WaitContainer    func(string) (int, error)
	RemoveImage      func(string) error
	CreateExec       func(docker.CreateExecOptions) (*docker.Exec, error)
	StartExec        func(string, docker.StartExecOptions) error
	Ping             func() error
	Stats            func(docker.StatsOptions) error
}

func NewDocker(endpoint, cert, key, ca string) *DockerWrapper {
	client, err := docker.NewTLSClient(endpoint, cert, key, ca)
	if err != nil {
		logs.Assert(err, "Docker")
	}
	d := &DockerWrapper{Client: client}
	var makeDockerWrapper func(*DockerWrapper, *docker.Client) *DockerWrapper
	MakeWrapper(&makeDockerWrapper)
	return makeDockerWrapper(d, client)
}
