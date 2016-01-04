package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime/pprof"

	_ "net/http/pprof"

	"github.com/HunanTV/eru-agent/app"
	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/lenz"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/HunanTV/eru-agent/network"
	"github.com/bmizerany/pat"
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
	type Endpoint struct {
		Nid int    `json:"nid"`
		IP  string `json:"ip"`
	}
	type Result struct {
		Succ        int    `json:"succ"`
		ContainerID string `json:"container_id"`
		VethName    string `json:"veth"`
		IP          string `json:"ip"`
	}

	cid := req.URL.Query().Get(":container_id")

	endpoints := []Endpoint{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&endpoints)
	if err != nil {
		return http.StatusBadRequest, JSON{"message": "wrong JSON format"}
	}

	rv := []Result{}
	for seq, endpoint := range endpoints {
		vethName := fmt.Sprintf("%s%d.%d", common.VLAN_PREFIX, endpoint.Nid, seq)
		if network.AddVLan(vethName, endpoint.IP, cid) {
			rv = append(rv, Result{Succ: 1, ContainerID: cid, VethName: vethName, IP: endpoint.IP})
		} else {
			rv = append(rv, Result{Succ: 0, ContainerID: "", VethName: "", IP: ""})
		}
	}
	return http.StatusOK, rv
}

// URL /api/container/:container_id/addcalico/
func addCalicoForContainer(req *Request) (int, interface{}) {
	type Endpoint struct {
		Nid     int    `json:"nid"`
		Profile string `json:"profile"`
		IP      string `json:"ip"`
		Append  bool   `json:"append"`
	}
	type Result struct {
		Succ        int    `json:"succ"`
		ContainerID string `json:"container_id"`
		IP          string `json:"ip"`
		Err         string `json:"err"`
	}

	if g.Config.VLan.Calico == "" {
		return http.StatusBadRequest, JSON{"message": "Agent not enable calico support"}
	}

	cid := req.URL.Query().Get(":container_id")
	env := os.Environ()
	env = append(env, fmt.Sprintf("ETCD_AUTHORITY=%s", g.Config.VLan.Calico))

	endpoints := []Endpoint{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&endpoints)
	if err != nil {
		return http.StatusBadRequest, JSON{"message": "wrong JSON format"}
	}

	rv := []Result{}
	for seq, endpoint := range endpoints {
		vethName := fmt.Sprintf("%s%d.%d", common.VLAN_PREFIX, endpoint.Nid, seq)
		if !endpoint.Append {
			add := exec.Command("calicoctl", "container", "add", cid, endpoint.IP, "--interface", vethName)
			add.Env = env
			if err := add.Run(); err != nil {
				rv = append(rv, Result{Succ: 0, ContainerID: cid, IP: endpoint.IP, Err: err.Error()})
				logs.Info("API calico add interface failed", err)
				continue
			}
		} else {
			add := exec.Command("calicoctl", "container", cid, "ip", "add", endpoint.IP, "--interface", vethName)
			add.Env = env
			if err := add.Run(); err != nil {
				rv = append(rv, Result{Succ: 0, ContainerID: cid, IP: endpoint.IP, Err: err.Error()})
				logs.Info("API calico append interface failed", err)
				continue
			}
		}

		//TODO remove when eru-core support ACL
		// currently only one profile is used
		profile := exec.Command("calicoctl", "container", cid, "profile", "append", endpoint.Profile)
		profile.Env = env
		if err := profile.Run(); err != nil {
			rv = append(rv, Result{Succ: 0, ContainerID: cid, IP: endpoint.IP, Err: err.Error()})
			logs.Info("API calico add profile failed", err)
			continue
		}

		rv = append(rv, Result{Succ: 1, ContainerID: cid, IP: endpoint.IP})
	}
	return http.StatusOK, rv
}

// URL /api/container/:container_id/addroute/
func addRouteForContainer(req *Request) (int, interface{}) {
	type Entry struct {
		CIDR      string
		Interface string
	}
	cid := req.URL.Query().Get(":container_id")
	data := &Entry{}

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(data); err != nil {
		return http.StatusBadRequest, JSON{"message": "wrong JSON format"}
	}
	if !network.AddRoute(cid, data.CIDR, data.Interface) {
		logs.Info("Add route failed")
		return http.StatusServiceUnavailable, JSON{"message": "add route failed"}
	}
	return http.StatusOK, JSON{"message": "ok"}
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
		if eruApp := app.NewEruApp(container, data.Meta); eruApp != nil {
			lenz.Attacher.Attach(&eruApp.Meta)
			app.Add(eruApp)
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
			"/api/container/add/":                     addNewContainer,
			"/api/container/:container_id/addvlan/":   addVlanForContainer,
			"/api/container/:container_id/addcalico/": addCalicoForContainer,
			"/api/container/:container_id/setroute/":  setRouteForContainer,
			"/api/container/:container_id/addroute/":  addRouteForContainer,
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
