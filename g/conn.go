package g

import (
	"fmt"

	"github.com/docker/engine-api/client"
	"github.com/keimoon/gore"
	"github.com/projecteru/eru-agent/defines"

	log "github.com/Sirupsen/logrus"
)

var Docker *client.Client
var Rds *gore.Pool
var ErrChan chan error

func InitialConn() {
	var err error
	Rds = &gore.Pool{
		InitialConn: Config.Redis.Min,
		MaximumConn: Config.Redis.Max,
	}

	redisHost := fmt.Sprintf("%s:%d", Config.Redis.Host, Config.Redis.Port)
	if err := Rds.Dial(redisHost); err != nil {
		log.Panicf("Init redis failed %s", err)
	}

	if Docker, err = defines.NewDocker(Config.Docker.Endpoint); err != nil {
		log.Panicf("Init docker cli failed %s", err)
	}

	log.Info("Global connections initiated")
	ErrChan = make(chan error)
}

func CloseConn() {
	Rds.Close()
	log.Info("Global connections closed")
}

func GetRedisConn() *gore.Conn {
	conn, err := Rds.Acquire()
	if err != nil || conn == nil {
		log.Panicf("Get redis connection failed %s", err)
	}
	return conn
}

func ReleaseRedisConn(conn *gore.Conn) {
	Rds.Release(conn)
}
