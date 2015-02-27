package metrics

import (
	"math/rand"
	"time"

	"../logs"
	"github.com/docker/libcontainer/cgroups"
)

func InitDevDir() {
	logs.Info("OSX not support cgroup mount point")
}

func GetCgroupStats(id string) (m *cgroups.Stats, err error) {
	logs.Info("OSX not support get cgroup stats", id)
	err = nil
	rand.Seed(time.Now().UnixNano())
	x := rand.Int63n(1e9)
	y := rand.Int63n(1e9)
	m = &cgroups.Stats{}
	m.CpuStats = cgroups.CpuStats{
		CpuUsage: cgroups.CpuUsage{
			TotalUsage:        uint64(x + y),
			UsageInUsermode:   uint64(x),
			UsageInKernelmode: uint64(y),
		},
	}
	s := map[string]uint64{}
	s["rss"] = uint64(rand.Int63n(9e10))
	m.MemoryStats = cgroups.MemoryStats{
		Usage: s["rss"] + uint64(rand.Int63n(9e10)),
		Stats: s,
	}
	return
}
