package g

import (
	"fmt"

	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/keimoon/gore"
)

var Docker defines.ContainerManager
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
	logs.Info("Global connections initiated")
}

func CloseConn() {
	Rds.Close()
	logs.Info("Global connections closed")
}

func GetRedisConn() *gore.Conn {
	conn, err := Rds.Acquire()
	if err != nil || conn == nil {
		logs.Assert(err, "Get redis conn")
	}
	return conn
}

func ReleaseRedisConn(conn *gore.Conn) {
	Rds.Release(conn)
}
