package lenz

import (
	"encoding/json"
	"fmt"
	"io"
	"log/syslog"
	"math"
	"net"
	"net/url"

	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/logs"
)

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
				//logs.Debug("Lenz Send", logline.Name, logline.EntryPoint, logline.ID, "to", addr)
				switch u, err := url.Parse(addr); {
				case err != nil:
					logs.Info("Lenz", err)
					route.Backends.Remove(addr)
					continue
				case u.Scheme == "udp":
					if err := udpStreamer(logline, u.Host); err != nil {
						logs.Info("Lenz Send to", u.Host, "by udp failed", err)
						continue
					}
				case u.Scheme == "tcp":
					if err := tcpStreamer(logline, u.Host); err != nil {
						logs.Info("Lenz Send to", u.Host, "by tcp failed", err)
						continue
					}
				case u.Scheme == "syslog":
					if err := syslogStreamer(logline, u.Host); err != nil {
						logs.Info("Lenz Sent to syslog failed", err)
						continue
					}
				}
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

func syslogStreamer(logline *defines.Log, addr string) error {
	tag := fmt.Sprintf("%s.%s", logline.Name, logline.Tag)
	remote, err := syslog.Dial("udp", addr, syslog.LOG_USER|syslog.LOG_INFO, tag)
	if err != nil {
		return err
	}
	_, err = io.WriteString(remote, logline.Data)
	return err
}

func tcpStreamer(logline *defines.Log, addr string) error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		logs.Debug("Resolve tcp failed", err)
		return err
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		logs.Debug("Connect backend failed", err)
		return err
	}
	defer conn.Close()
	return writeJSON(conn, logline)
}

func udpStreamer(logline *defines.Log, addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		logs.Debug("Resolve udp failed", err)
		return err
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		logs.Debug("Connect backend failed", err)
		return err
	}
	defer conn.Close()
	return writeJSON(conn, logline)
}

func writeJSON(w io.Writer, logline *defines.Log) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(logline)
}
