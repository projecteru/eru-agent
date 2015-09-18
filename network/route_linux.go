package network

import (
	"os/exec"
	"strconv"

	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
)

func SetDefaultRoute(cid, gateway string) bool {
	lock.Lock()
	defer lock.Unlock()
	logs.Info("Set", cid[:12], "default route", gateway)

	container, err := g.Docker.InspectContainer(cid)
	if err != nil {
		logs.Info("RouteSetter inspect docker failed", err)
		return false
	}

	pid := strconv.Itoa(container.State.Pid)
	cmd := exec.Command("nsenter", "-t", pid, "-n", "route", "del", "default")
	if err := cmd.Run(); err != nil {
		logs.Info("Clean default route failed", err)
		return false
	}

	cmd = exec.Command("nsenter", "-t", pid, "-n", "route", "add", "default", "gw", gateway)
	if err := cmd.Run(); err != nil {
		logs.Info("RouteSetter set default route failed", err)
		return false
	}

	logs.Info("Set default route success", cid[:12], gateway)
	return true
}
