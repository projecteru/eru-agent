package network

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
)

var calicoctl = "/usr/local/bin/calicoctl"

func StartCalicoNode() bool {
	lock.Lock()
	defer lock.Unlock()

	logs.Info("Start a a calico node on host", g.Config.Calico.NodeIP)

	ip := fmt.Sprintf("--ip=%s", g.Config.Calico.NodeIP)
	node_image := fmt.Sprintf("--node-image=%s", g.Config.Calico.NodeImage)

	logs.Debug(os.Environ())
	logs.Debug("running command: ", calicoctl, "node", ip, node_image)
	out, err := exec.Command(calicoctl, "node", ip, node_image).CombinedOutput()
	if err != nil {
		logs.Info("Start node on host failed", err)
		logs.Debug(fmt.Sprintf("%s", out))
		return false
	}
	logs.Info(fmt.Sprintf("%s", out))
	return true
}

func StopCalicoNode() bool {
	lock.Lock()
	defer lock.Unlock()

	logs.Info("stop calico node on host", g.Config.HostName)
	out, err := exec.Command(calicoctl, "node", "stop", "--force").CombinedOutput()
	if err != nil {
		logs.Info("node stop failed ", err)
		logs.Debug(fmt.Sprintf("%s", out))
		return false
	}
	logs.Debug(fmt.Sprintf("%s", out))
	return true
}

func AddContaienrToCalicoNet(containerID, ip string) bool {
	lock.Lock()
	defer lock.Unlock()

	logs.Info("adding container", containerID, "to calico network ...")
	cmd := exec.Command(calicoctl, "container", "add", containerID, ip)
	logs.Debug(cmd.Args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logs.Info("add contaienr ", containerID, "to calico network failed", err)
		logs.Debug(fmt.Sprintf("%s", out))
		return false
	}
	logs.Info("add container to network successfully")
	return true
}

func RemoveContainerFromCalicoNet(containerID string) bool {
	lock.Lock()
	defer lock.Unlock()

	logs.Info("removing container", containerID, "from calico network ...")
	cmd := exec.Command(calicoctl, "container", "remove", containerID)
	logs.Debug(cmd.Args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logs.Info("remove container ", containerID, "from calico network fail", err)
		logs.Debug(fmt.Sprintf("%s", out))
		return false
	}
	logs.Info("remove container", containerID, "from calico network successfully ")
	return true
}

func ContaienrIP(containerID, action, ip string) bool {
	lock.Lock()
	defer lock.Unlock()

	logs.Info(containerID, action, ip)
	cmd := exec.Command(calicoctl, "container", containerID, "ip", action, ip)
	logs.Debug(cmd.Args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logs.Info(action, containerID, "ip:", ip, " failed")
		logs.Debug(fmt.Sprintf("%s", out))
		return false
	}

	logs.Info(action, containerID, "ip successfully")
	return true
}

func ShowContainerEndPointId(containerID string) (string, error) {
	logs.Info("show container", containerID, "'s endpoint id")
	cmd := exec.Command(calicoctl, "container", containerID, "endpoint-id", "show")
	logs.Debug(cmd.Args)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(fmt.Sprintf("%s", out)), err
}
