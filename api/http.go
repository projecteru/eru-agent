package api

import (
	"fmt"
	"net/http"
	"runtime/pprof"

	_ "net/http/pprof"

	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/bmizerany/pat"
)

func version(req *Request) interface{} {
	return JSON{
		"version": common.VERSION,
	}
}

func list(req *Request) interface{} {
	ret := JSON{}
	for ID, EruApp := range g.Apps {
		ret[ID] = EruApp.Meta
	}
	return ret
}

func addApp(req *Request) interface{} {
	fmt.Println(req.Body)
	fmt.Println(req.Form)
	fmt.Println(req.URL.Query())
	return JSON{}
}

func profile(req *Request) interface{} {
	r := JSON{}
	for _, p := range pprof.Profiles() {
		r[p.Name()] = p.Count()
	}
	return r
}

func HTTPServe() {
	m := pat.New()
	m.Add("GET", "/profile", http.HandlerFunc(JSONWrapper(profile)))
	m.Add("GET", "/", http.HandlerFunc(JSONWrapper(version)))
	m.Add("GET", "/api/list", http.HandlerFunc(JSONWrapper(list)))
	m.Add("PUT", "/api/add", http.HandlerFunc(JSONWrapper(addApp)))

	http.Handle("/", m)
	logs.Info("API http server start at", g.Config.API.Addr)
	err := http.ListenAndServe(g.Config.API.Addr, nil)
	if err != nil {
		logs.Info(err, "ListenAndServe: ")
	}
}
