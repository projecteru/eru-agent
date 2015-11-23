package defines

import (
	"github.com/fsouza/go-dockerclient"
)

type ContainerManager interface {
	PushImage(docker.PushImageOptions, docker.AuthConfiguration) error
	PullImage(docker.PullImageOptions, docker.AuthConfiguration) error
	CreateContainer(docker.CreateContainerOptions) (*docker.Container, error)
	StartContainer(string, *docker.HostConfig) error
	BuildImage(docker.BuildImageOptions) error
	KillContainer(docker.KillContainerOptions) error
	StopContainer(string, uint) error
	InspectContainer(string) (*docker.Container, error)
	ListContainers(docker.ListContainersOptions) ([]docker.APIContainers, error)
	ListImages(docker.ListImagesOptions) ([]docker.APIImages, error)
	RemoveContainer(docker.RemoveContainerOptions) error
	WaitContainer(string) (int, error)
	RemoveImage(string) error
	CreateExec(docker.CreateExecOptions) (*docker.Exec, error)
	StartExec(string, docker.StartExecOptions) error
	Ping() error
	Stats(docker.StatsOptions) error
	AttachToContainer(opts docker.AttachToContainerOptions) error
	AddEventListener(listener chan<- *docker.APIEvents) error
}

func NewDocker(endpoint, cert, key, ca string) (ContainerManager, error) {
	client, err := docker.NewTLSClient(endpoint, cert, key, ca)
	if err != nil {
		return nil, err
	}
	return client, nil
}
