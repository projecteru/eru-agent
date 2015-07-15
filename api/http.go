package api

import (
	"net/http"

	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/HunanTV/eru-agent/status"
	"github.com/bmizerany/pat"
)

func version(req *Request) interface{} {
	return JSON{
		"version": common.VERSION,
	}
}

func listStatus(req *Request) interface{} {
	return JSON{
		"status": status.Apps,
	}
}

func addContainer(req *Request) interface{} {
	return JSON{
		"status": status.Apps,
	}
}

func HTTPServe() {
	m := pat.New()
	m.Add("GET", "/", http.HandlerFunc(JSONWrapper(version)))
	m.Add("GET", "/api/status/list", http.HandlerFunc(JSONWrapper(listStatus)))
	m.Add("PUT", "/api/add", http.HandlerFunc(JSONWrapper(addContainer)))

	http.Handle("/", m)
	logs.Info("API http server start at", g.Config.API.Addr)
	err := http.ListenAndServe(g.Config.API.Addr, nil)
	if err != nil {
		logs.Info(err, "ListenAndServe: ")
	}
}
