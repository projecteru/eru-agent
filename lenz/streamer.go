package lenz

import (
	"math"

	"github.com/projecteru/eru-agent/defines"
	"github.com/projecteru/eru-agent/g"

	log "github.com/Sirupsen/logrus"
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
		log.Debugf("Flush %s cache logs", route.ID)
		for _, remote := range upstreams {
			remote.Flush()
			for _, msg := range remote.Tail() {
				log.Infof("Streamer can't send to remote %s", msg)
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
			log.Infof("Debug Output %v", logline)
			continue
		}
		var f bool = false
		for offset := 0; offset < route.Backends.Len(); offset++ {
			addr := route.Backends.Get(logline.Name, offset)
			if _, ok := upstreams[addr]; !ok {
				if ups, err := NewUpStream(addr); err != nil || ups == nil {
					continue
				} else {
					upstreams[addr] = ups
				}
			}
			f = true
			if err := upstreams[addr].WriteData(logline); err != nil {
				log.Errorf("Sent to remote failed %s", err)
				upstreams[addr].Close()
				go func(upstream *UpStream) {
					for _, log := range upstream.Tail() {
						logstream <- log
					}
				}(upstreams[addr])
				delete(upstreams, addr)
				continue
			}
			//log.Debugf("Lenz Send %s %s %s to %s", logline.Name, logline.EntryPoint, logline.ID, addr)
			break
		}
		if !f {
			log.Infof("Lenz failed %s %s %s %s", logline.ID[:12], logline.Name, logline.EntryPoint, logline.Data)
		}
		if count == math.MaxInt64 {
			count = 0
		} else {
			count++
		}
	}
}
