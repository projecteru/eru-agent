package defines

import (
	"time"

	"github.com/fsouza/go-dockerclient"
)

type Metric struct {
	Step     time.Duration
	Client   SingleConnRpcClient
	Tag      string
	Endpoint string

	Last time.Time
	Exec *docker.Exec

	Stop chan bool
	Info map[string]uint64
	Save map[string]uint64
	Rate map[string]float64
}
