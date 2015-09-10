package defines

import "time"

type Metric struct {
	Step     time.Duration
	Client   SingleConnRpcClient
	Tag      string
	Endpoint string
	Last     time.Time

	Stop chan bool
	Info map[string]uint64
	Save map[string]uint64
	Rate map[string]float64
}
