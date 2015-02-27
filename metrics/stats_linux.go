package metrics

import (
	"../logs"

	"github.com/docker/libcontainer/cgroups"
	"github.com/docker/libcontainer/cgroups/fs"
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
	if parentName, id, _, err = getLongID(id); err != nil {
		return
	}
	c := cgroups.Cgroup{
		Parent: parentName,
		Name:   id,
	}
	return fs.GetStats(&c)
}
