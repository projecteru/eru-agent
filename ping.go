package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
)

func Ping() {
	ticker := time.Tick(time.Duration(g.Config.Docker.Health) * time.Second)
	for _ = range ticker {
		if err := g.Docker.Ping(); err != nil {
			url := fmt.Sprintf("%s/api/host/%s/down", g.Config.Eru.Endpoint, g.Config.HostName)
			client := &http.Client{}
			req, _ := http.NewRequest("PUT", url, nil)
			client.Do(req)
			logs.Assert(err, "Docker exit")
		}
	}
}
