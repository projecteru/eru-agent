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

// URL /version/
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

// URL /api/container/:container_id/addvlan/
func addVlanForContainer(req *Request) (int, interface{}) {
	type IP struct {
		Nid int    `json:"nid"`
		IP  string `json:"ip"`
	}
	type Data struct {
		TaskID string `json:"task_id"`
		IPs    []IP   `json:"ips"`
	}

	conn := g.GetRedisConn()
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
		vethName := fmt.Sprintf("%s%d.%d", common.VLAN_PREFIX, ip.Nid, seq)
		if network.AddVLan(vethName, ip.IP, cid) {
			gore.NewCommand("LPUSH", feedKey, fmt.Sprintf("1|%s|%s|%s", cid, vethName, ip.IP)).Run(conn)
			continue
		} else {
			gore.NewCommand("LPUSH", feedKey, "0|||").Run(conn)
		}
	}
	return http.StatusOK, JSON{"message": "ok"}
}

// URL POST /api/calico/node/
func startCalicoNode(req *Request) (int, interface{}) {
	if network.StartCalicoNode() {
		return http.StatusCreated, JSON{"message": "create successful"}
	} else {
		return http.StatusBadRequest, JSON{"message": "calico node start fail"}
	}
}

// URL DELETE /api/calico/node/
func stopCalicoNode(req *Request) (int, interface{}) {
	if network.StopCalicoNode() {
		return http.StatusAccepted, JSON{"message": "stopn successful"}
	} else {
		return http.StatusBadRequest, JSON{"message": "calico node stop fail"}
	}
}

//URL POST /api/calico/container/:container_id/
func addContainerToCalicoNet(req *Request) (int, interface{}) {
	type Data struct {
		IPAddr string `json:"ip_addr"`
	}

	container_id := req.URL.Query().Get(":container_id")

	logs.Debug("Request", req)
	logs.Debug("body", req.Body)
	data := &Data{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(data)
	if err != nil {
		return http.StatusBadRequest, JSON{"message": "wrong JSON format"}
	}
	logs.Debug("received json: ", data)
	if network.AddContaienrToCalicoNet(container_id, data.IPAddr) {
		return http.StatusCreated, JSON{"message": "add container to calico success"}
	} else {
		return http.StatusBadRequest, JSON{"message": "add container to calico fail"}
	}
}

//URL DELETE /api/calico/container/:container_id/
func deleteContainerFromCalicoNet(req *Request) (int, interface{}) {
	container_id := req.URL.Query().Get(":container_id")

	if network.RemoveContainerFromCalicoNet(container_id) {
		return http.StatusAccepted, JSON{"message": "remove container to calico success"}
	} else {
		return http.StatusBadRequest, JSON{"message": "remove container to calico fail"}
	}
}

//URL GET /api/calico/container/:container_id/endpoint/
func showEndPointForContainer(req *Request) (int, interface{}) {
	container_id := req.URL.Query().Get(":container_id")
	out, err := network.ShowContainerEndPointId(container_id)
	if err != nil {
		logs.Debug("get container's endpoint id fail", err)
		return http.StatusNotFound, JSON{"message": err.Error()}
	}
	return http.StatusOK, JSON{"endpoint_id": out}
}

//URL POST /api/calico/containerip/:container_id/
func addIPToContainer(req *Request) (int, interface{}) {
	type Data struct {
		IPAddr string `json:"ip_addr"`
	}

	container_id := req.URL.Query().Get(":container_id")

	data := &Data{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(data)
	if err != nil {
		return http.StatusBadRequest, JSON{"message": "wrong JSON format"}
	}

	if network.ContaienrIP(container_id, "add", data.IPAddr) {
		return http.StatusCreated, JSON{"message": "create ip success"}
	} else {
		return http.StatusBadRequest, JSON{"message": "create ip to container fail"}
	}
}

//URL DELETE /api/calico/containerip/:container_id/
func removeIPFromContainer(req *Request) (int, interface{}) {
	type Data struct {
		IPAddr string `json:"ip_addr"`
	}

	container_id := req.URL.Query().Get(":container_id")

	data := &Data{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(data)
	if err != nil {
		return http.StatusBadRequest, JSON{"message": "wrong JSON format"}
	}

	if network.ContaienrIP(container_id, "remove", data.IPAddr) {
		return http.StatusCreated, JSON{"message": "create ip success"}
	} else {
		return http.StatusBadRequest, JSON{"message": "create ip to container fail"}
	}
}

// URL /api/container/:container_id/setroute/
func setRouteForContainer(req *Request) (int, interface{}) {
	type Gateway struct {
		IP string
	}
	cid := req.URL.Query().Get(":container_id")
	data := &Gateway{}

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(data); err != nil {
		return http.StatusBadRequest, JSON{"message": "wrong JSON format"}
	}
	if !network.SetDefaultRoute(cid, data.IP) {
		logs.Info("Set default route failed")
		return http.StatusServiceUnavailable, JSON{"message": "set default route failed"}
	}
	return http.StatusOK, JSON{"message": "ok"}
}

// URL /api/container/add/
func addNewContainer(req *Request) (int, interface{}) {
	type Data struct {
		Control     string                 `json:"control"`
		ContainerID string                 `json:"container_id"`
		Meta        map[string]interface{} `json:"meta"`
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
			"/profile/":                                     profile,
			"/version/":                                     version,
			"/api/app/list/":                                listEruApps,
			"/api/calico/container/:container_id/endpoint/": showEndPointForContainer,
		},
		"POST": {
			"/api/container/add/":                    addNewContainer,
			"/api/container/:container_id/addvlan/":  addVlanForContainer,
			"/api/container/:container_id/setroute/": setRouteForContainer,
			"/api/calico/node/":                      startCalicoNode,
			"/api/calico/container/:container_id/":   addContainerToCalicoNet,
			"/api/calico/containerip/:container_id/": addIPToContainer,
		},
		"DELETE": {
			"/api/calico/node/":                      stopCalicoNode,
			"/api/calico/container/:container_id/":   deleteContainerFromCalicoNet,
			"/api/calico/containerip/:container_id/": removeIPFromContainer,
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
