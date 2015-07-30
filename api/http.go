package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/pprof"

	_ "net/http/pprof"

	"github.com/HunanTV/eru-agent/app"
	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/lenz"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/HunanTV/eru-agent/network"
	"github.com/bmizerany/pat"
	"github.com/keimoon/gore"
)

// 给跪了
// TODO 之后把这个给抽出去吧
func getRedisConn() *gore.Conn {
	conn, err := g.Rds.Acquire()
	if err != nil || conn == nil {
		logs.Assert(err, "Get redis conn")
	}
	return conn
}

// URL /api/version/
func version(req *Request) (int, interface{}) {
	return http.StatusOK, JSON{"version": common.VERSION}
}

// URL /profile/
func profile(req *Request) (int, interface{}) {
	r := JSON{}
	for _, p := range pprof.Profiles() {
		r[p.Name()] = p.Count()
	}
	return http.StatusOK, r
}

// URL /api/app/list/
func listEruApps(req *Request) (int, interface{}) {
	ret := JSON{}
	for ID, EruApp := range app.Apps {
		ret[ID] = EruApp.Meta
	}
	return http.StatusOK, ret
}

// URL /api/app/add/
func addEruApp(req *Request) (int, interface{}) {
	fmt.Println(req.Body)
	fmt.Println(req.Form)
	fmt.Println(req.URL.Query())
	return http.StatusOK, JSON{}
}

// URL /api/container/:container_id/addvlan/
func addVlanForContainer(req *Request) (int, interface{}) {
	type IP struct {
		Nid int    `json: "nid"`
		IP  string `json: "ip"`
	}
	type Data struct {
		TaskID int  `json: "task_id"`
		IPs    []IP `json: "ips"`
	}

	conn := getRedisConn()
	defer g.Rds.Release(conn)

	cid := req.URL.Query().Get(":container_id")

	data := &Data{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(data)
	if err != nil {
		return http.StatusBadRequest, JSON{"message": "wrong JSON format"}
	}

	feedKey := fmt.Sprintf("eru:agent:%s:feedback", data.TaskID)
	for seq, ip := range data.IPs {
		vethName := fmt.Sprintf("%s%s.%d", common.VLAN_PREFIX, ip.Nid, seq)
		if network.AddVLan(vethName, ip.IP, cid) {
			gore.NewCommand("LPUSH", feedKey, fmt.Sprintf("1|%s|%s|%s", cid, vethName, ip.IP)).Run(conn)
			continue
		} else {
			gore.NewCommand("LPUSH", feedKey, "0|||").Run(conn)
		}
	}
	return http.StatusOK, JSON{"message": "ok"}
}

// URL /api/container/add/
func addNewContainer(req *Request) (int, interface{}) {
	type Data struct {
		Control     string                 `json: "control"`
		ContainerID string                 `json: "container_id"`
		Meta        map[string]interface{} `json: "meta"`
	}

	data := &Data{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(data)
	if err != nil {
		return http.StatusBadRequest, JSON{"message": "wrong JSON format"}
	}

	switch data.Control {
	case "+":
		if app.Valid(data.ContainerID) {
			break
		}
		logs.Info("API status watch", data.ContainerID)
		container, err := g.Docker.InspectContainer(data.ContainerID)
		if err != nil {
			logs.Info("API status inspect docker failed", err)
			break
		}
		if eruApp := app.NewEruApp(container.ID, container.Name, data.Meta); eruApp != nil {
			app.Add(eruApp)
			lenz.Attacher.Attach(&eruApp.Meta)
		}
	}
	return http.StatusOK, JSON{"message": "ok"}
}

func HTTPServe() {
	restfulAPIServer := pat.New()

	handlers := map[string]map[string]func(*Request) (int, interface{}){
		"GET": {
			"/profile/":      profile,
			"/version/":      version,
			"/api/app/list/": listEruApps,
		},
		"POST": {
			"/api/container/add/":                   addNewContainer,
			"/api/container/:container_id/addvlan/": addVlanForContainer,
			"/api/app/add/":                         addEruApp,
		},
	}

	for method, routes := range handlers {
		for route, handler := range routes {
			restfulAPIServer.Add(method, route, http.HandlerFunc(JSONWrapper(handler)))
		}
	}

	http.Handle("/", restfulAPIServer)
	logs.Info("API http server start at", g.Config.API.Addr)
	err := http.ListenAndServe(g.Config.API.Addr, nil)
	if err != nil {
		logs.Assert(err, "ListenAndServe: ")
	}
}
