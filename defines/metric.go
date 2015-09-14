package defines

import "time"

type Metric struct {
	Step     time.Duration
	Client   SingleConnRpcClient
	Tag      string
	Endpoint string
	Last     time.Time

	Stop chan bool
	Save map[string]uint64
}
