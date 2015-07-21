package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/HunanTV/eru-agent/app"
	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/lenz"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/HunanTV/eru-agent/network"
	"github.com/keimoon/gore"
)

func PubSubServe() {
	go statusWatcher()
	go vlanWatcher()
}

func getRdsConn() *gore.Conn {
	conn, err := g.Rds.Acquire()
	if err != nil || conn == nil {
		logs.Assert(err, "Get redis conn")
	}
	return conn
}

func releaseConn(conn *gore.Conn) {
	g.Rds.Release(conn)
}

func vlanWatcher() {
	conn := getRdsConn()
	report := getRdsConn()
	defer releaseConn(conn)
	defer releaseConn(report)

	subs := gore.NewSubscriptions(conn)
	defer subs.Close()
	subKey := fmt.Sprintf("eru:agent:%s:vlan", g.Config.HostName)
	logs.Debug("API vlan subscribe", subKey)
	subs.Subscribe(subKey)

	for message := range subs.Message() {
		if message == nil {
			logs.Info("API vLan watcher shutdown")
			break
		}
		command := string(message.Message)
		logs.Debug("API vlan watcher get", command)
		parser := strings.Split(command, "|")
		if len(parser) <= 3 {
			logs.Info("API vlan watcher command invaild", command)
			continue
		}
		taskID, containerID := parser[0], parser[1]
		feedKey := fmt.Sprintf("eru:agent:%s:feedback", taskID)
		for _, content := range parser[2:] {
			p := strings.Split(content, ":")
			if len(p) != 2 {
				logs.Info("API vlan watcher command invaild", content)
				continue
			}
			seq, ips := p[0], p[1]
			vethName := fmt.Sprintf("%s%s", common.VLAN_PREFIX, seq)
			if network.AddVLan(vethName, ips, containerID) {
				gore.NewCommand("LPUSH", feedKey, fmt.Sprintf("1|%s|%s|%s", containerID, vethName, ips)).Run(report)
				continue
			}
			gore.NewCommand("LPUSH", feedKey, fmt.Sprintf("0|||")).Run(report)
		}
	}
}

func statusWatcher() {
	conn := getRdsConn()
	defer releaseConn(conn)

	subs := gore.NewSubscriptions(conn)
	defer subs.Close()
	subKey := fmt.Sprintf("eru:agent:%s:watcher", g.Config.HostName)
	logs.Debug("API status subscribe", subKey)
	subs.Subscribe(subKey)

	for message := range subs.Message() {
		if message == nil {
			logs.Info("API status watcher shutdown")
			break
		}
		command := string(message.Message)
		logs.Debug("API status watcher get", command)
		parser := strings.Split(command, "|")
		control, containerID, metaString := parser[0], parser[1], parser[2]
		switch control {
		case "+":
			if app.Vaild(containerID) {
				break
			}
			logs.Info("API status watch", containerID)
			container, err := g.Docker.InspectContainer(containerID)
			if err != nil {
				logs.Info("API status inspect docker failed", err)
				break
			}
			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(metaString), &meta); err != nil {
				logs.Info("API status load failed", err)
				break
			}
			if eruApp := app.NewEruApp(container.ID, container.Name, meta); eruApp != nil {
				app.Add(eruApp)
				lenz.Attacher.Attach(&eruApp.Meta)
			}
		}
	}
}
