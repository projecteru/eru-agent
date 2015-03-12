package metrics

import (
	"testing"

	"github.com/fsouza/go-dockerclient"

	"../common"
	"../defines"
)

var Metrics *MetricsRecorder
var config defines.MetricsConfig

func init() {
	common.Docker = defines.NewDocker("tcp://192.168.59.103:2375")
	defines.MockDocker(common.Docker)
	config = defines.MetricsConfig{10, "localhost:8083", "root", "root", "test"}
	Metrics = NewMetricsRecorder("test", config)
}

func Test_MetricReporter(t *testing.T) {
	cid := "123"
	common.Docker.CreateExec = func(docker.CreateExecOptions) (*docker.Exec, error) {
		return &docker.Exec{"123"}, nil
	}
	common.Docker.StartExec = func(id string, opt docker.StartExecOptions) error {
		opt.Success <- struct{}{}
		<-opt.Success
		return nil
	}
	Metrics.Add(cid, &defines.App{"test", "raw", "raw_app"})
	if _, ok := Metrics.apps[cid]; !ok {
		t.Error("Add Failed")
	}
	Metrics.Remove(cid)
	if _, ok := Metrics.apps[cid]; ok {
		t.Error("Remove Failed")
	}
}
