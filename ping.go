package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/logs"
)

func Ping() {
	ticker := time.Tick(time.Duration(config.Docker.Health) * time.Second)
	for _ = range ticker {
		if err := common.Docker.Ping(); err != nil {
			url := fmt.Sprintf("%s/api/host/%s/down", config.Eru.Endpoint, config.HostName)
			client := &http.Client{}
			req, _ := http.NewRequest("PUT", url, nil)
			client.Do(req)
			logs.Assert(err, "Docker exit")
		}
	}
}
