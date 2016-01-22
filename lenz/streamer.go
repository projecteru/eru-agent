package lenz

import (
	"math"

	"github.com/projecteru/eru-agent/defines"
	"github.com/projecteru/eru-agent/g"
	"github.com/projecteru/eru-agent/logs"
)

func Streamer(route *defines.Route, logstream chan *defines.Log) {
	var upstreams map[string]*UpStream = map[string]*UpStream{}
	var types map[string]struct{}
	var count int64 = 0
	if route.Source != nil {
		types = make(map[string]struct{})
		for _, t := range route.Source.Types {
			types[t] = struct{}{}
		}
	}
	defer func() {
		logs.Debug("Flush", route.ID, "cache logs")
		for _, remote := range upstreams {
			remote.Flush()
			for _, log := range remote.Tail() {
				logs.Info("Streamer can't send to remote", log)
			}
			remote.Close()
		}
		route.Done <- struct{}{}
	}()
	for logline := range logstream {
		if types != nil {
			if _, ok := types[logline.Type]; !ok {
				continue
			}
		}
		logline.Tag = route.Target.AppendTag
		logline.Count = count
		if g.Config.Lenz.Stdout {
			logs.Info("Debug Output", logline)
			continue
		}
		var f bool = false
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
			f = true
			if err := upstreams[addr].WriteData(logline); err != nil {
				logs.Info("Sent to remote failed", err)
				upstreams[addr].Close()
				go func(upstream *UpStream) {
					for _, log := range upstream.Tail() {
						logstream <- log
					}
				}(upstreams[addr])
				delete(upstreams, addr)
				continue
			}
			//logs.Debug("Lenz Send", logline.Name, logline.EntryPoint, logline.ID, "to", addr)
			break
		}
		if !f {
			logs.Info("Lenz failed", logline.ID[:12], logline.Name, logline.EntryPoint, logline.Data)
		}
		if count == math.MaxInt64 {
			count = 0
		} else {
			count++
		}
	}
}
