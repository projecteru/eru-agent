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
	go routeWatcher()
}

func vlanWatcher() {
	conn := g.GetRedisConn()
	report := g.GetRedisConn()
	defer g.ReleaseRedisConn(conn)
	defer g.ReleaseRedisConn(report)

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
		if len(parser) <= 2 {
			logs.Info("API vlan watcher command invaild", command)
			continue
		}
		taskID, cid := parser[0], parser[1]
		feedKey := fmt.Sprintf("eru:agent:%s:feedback", taskID)
		for seq, content := range parser[2:] {
			p := strings.Split(content, ":")
			if len(p) != 2 {
				logs.Info("API vlan watcher ips invaild", content)
				continue
			}
			nid, ips := p[0], p[1]
			vethName := fmt.Sprintf("%s%s.%d", common.VLAN_PREFIX, nid, seq)
			if network.AddVLan(vethName, ips, cid) {
				gore.NewCommand("LPUSH", feedKey, fmt.Sprintf("1|%s|%s|%s", cid, vethName, ips)).Run(report)
				continue
			}
			gore.NewCommand("LPUSH", feedKey, "0|||").Run(report)
		}
	}
}

func routeWatcher() {
	conn := g.GetRedisConn()
	defer g.ReleaseRedisConn(conn)

	subs := gore.NewSubscriptions(conn)
	defer subs.Close()
	subKey := fmt.Sprintf("eru:agent:%s:route", g.Config.HostName)
	logs.Debug("API route subscribe", subKey)
	subs.Subscribe(subKey)

	for message := range subs.Message() {
		if message == nil {
			logs.Info("API route watcher shutdown")
			break
		}
		command := string(message.Message)
		logs.Debug("API route watcher get", command)
		parser := strings.Split(command, "|")
		if len(parser) != 2 {
			logs.Info("API route watcher command invaild", command)
			continue
		}
		cid, gateway := parser[0], parser[1]
		if !network.SetDefaultRoute(cid, gateway) {
			logs.Info("Set default route failed")
		}
	}
}

func statusWatcher() {
	conn := g.GetRedisConn()
	defer g.ReleaseRedisConn(conn)

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
		if len(parser) != 3 {
			logs.Info("API status watcher command invaild", command)
			continue
		}
		control, cid, metaString := parser[0], parser[1], parser[2]
		switch control {
		case "+":
			if app.Valid(cid) {
				break
			}
			logs.Info("API status watch", cid[:12])
			container, err := g.Docker.InspectContainer(cid)
			if err != nil {
				logs.Info("API status inspect docker failed", err)
				break
			}
			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(metaString), &meta); err != nil {
				logs.Info("API status load failed", err)
				break
			}
			if eruApp := app.NewEruApp(container, meta); eruApp != nil {
				lenz.Attacher.Attach(&eruApp.Meta)
				app.Add(eruApp)
			}
		}
	}
}
