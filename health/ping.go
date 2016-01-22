package health

import (
	"fmt"
	"time"

	"github.com/projecteru/eru-agent/g"
	"github.com/projecteru/eru-agent/logs"
	"github.com/projecteru/eru-agent/utils"
)

func Check() {
	go ping()
}

func ping() {
	ticker := time.Tick(time.Duration(g.Config.Docker.Health) * time.Second)
	for _ = range ticker {
		if err := g.Docker.Ping(); err != nil {
			url := fmt.Sprintf("%s/api/host/%s/down/", g.Config.Eru.Endpoint, g.Config.HostName)
			utils.DoPut(url)
			logs.Assert(err, "Docker exit")
		}
	}
}
