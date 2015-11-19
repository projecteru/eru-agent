package defines

import "time"

type Remote interface {
	Send(data map[string]float64, endpoint, tag string, timestamp, step int64) error
}

type Metric struct {
	Step     time.Duration
	Client   Remote
	Tag      string
	Endpoint string
	Last     time.Time

	Stop chan bool
	Save map[string]uint64
}
