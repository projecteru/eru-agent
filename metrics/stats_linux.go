package metrics

import (
	"../logs"

	"github.com/docker/libcontainer/cgroups"
	"github.com/docker/libcontainer/cgroups/fs"
	"github.com/docker/libcontainer/configs"
)

func InitDevDir() {
	var err error
	devDir, err = cgroups.FindCgroupMountpoint("devices")
	if err != nil {
		return
	}
	logs.Debug("Device Dir", devDir)
}

func GetCgroupStats(id string) (m *cgroups.Stats, err error) {
	var parentName string
	var pid int
	if parentName, id, pid, err = getLongID(id); err != nil {
		return
	}
	c := configs.Cgroup{
		Parent: parentName,
		Name:   id,
	}
	manager := fs.Manager{&c, map[string]string{}}
	manager.Apply(pid)
	return manager.GetStats()
}
