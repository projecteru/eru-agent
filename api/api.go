package api

import (
	"net/http"

	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/defines"
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
		"status": status.Status.Apps,
	}
}

func HTTPServe(config defines.APIConfig) {
	m := pat.New()
	m.Add("GET", "/", http.HandlerFunc(JSONWrapper(version)))
	m.Add("GET", "/api/status", http.HandlerFunc(JSONWrapper(listStatus)))

	http.Handle("/", m)
	logs.Info("Start HTTP API server at", config.Addr)
	err := http.ListenAndServe(config.Addr, nil)
	if err != nil {
		logs.Info(err, "ListenAndServe: ")
	}
}
