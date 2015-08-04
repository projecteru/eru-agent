package lenz

import (
	"math"

	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/logs"
)

var upstreams map[string]*UpStream = map[string]*UpStream{}

func Streamer(route *defines.Route, logstream chan *defines.Log, stdout bool) {
	var types map[string]struct{}
	var count int64 = 0
	if route.Source != nil {
		types = make(map[string]struct{})
		for _, t := range route.Source.Types {
			types[t] = struct{}{}
		}
	}
	for logline := range logstream {
		if types != nil {
			if _, ok := types[logline.Type]; !ok {
				continue
			}
		}
		logline.Tag = route.Target.AppendTag
		logline.Count = count

		switch stdout {
		case true:
			logs.Info("Debug Output", logline)
		default:
			for offset := 0; offset < route.Backends.Len(); offset++ {
				addr, err := route.Backends.Get(logline.Name, offset)
				if err != nil {
					logs.Info("Get backend failed", err, logline.Name, logline.Data)
					break
				}
				if _, ok := upstreams[addr]; !ok {
					if ups, err := NewUpStream(addr); err != nil || ups == nil {
						route.Backends.Remove(addr)
						continue
					} else {
						upstreams[addr] = ups
					}
				}
				if err := upstreams[addr].WriteData(logline); err != nil {
					upstreams[addr].Close()
					delete(upstreams, addr)
					continue
				}
				//logs.Debug("Lenz Send", logline.Name, logline.EntryPoint, logline.ID, "to", addr)
				break
			}
		}
		if count == math.MaxInt64 {
			count = 0
		} else {
			count++
		}
	}
}
