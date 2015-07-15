package g

import (
	"fmt"

	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/keimoon/gore"
)

var Docker *defines.DockerWrapper
var Rds *gore.Pool

func InitialConn() {
	var err error
	Rds = &gore.Pool{
		InitialConn: Config.Redis.Min,
		MaximumConn: Config.Redis.Max,
	}

	redisHost := fmt.Sprintf("%s:%d", Config.Redis.Host, Config.Redis.Port)
	if err := Rds.Dial(redisHost); err != nil {
		logs.Assert(err, "Redis init failed")
	}

	if Docker, err = defines.NewDocker(
		Config.Docker.Endpoint,
		Config.Docker.Cert,
		Config.Docker.Key,
		Config.Docker.Ca,
	); err != nil {
		logs.Assert(err, "Docker")
	}
}

func CloseConn() {
	Rds.Close()
}
