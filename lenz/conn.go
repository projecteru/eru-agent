package lenz

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/syslog"
	"net"
	"net/url"

	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
)

type UpStream struct {
	addr    string
	scheme  string
	conn    io.Writer
	encoder *json.Encoder
	buffer  []*defines.Log
	count   int
	Close   func() error
}

func NewUpStream(addr string) (up *UpStream, err error) {
	u, err := url.Parse(addr)
	if err != nil {
		logs.Info("Parse upstream addr failed", err)
		return nil, err
	}
	up = &UpStream{addr: u.Host}
	up.buffer = []*defines.Log{}
	up.count = 0
	switch {
	case u.Scheme == "udp":
		err = up.createUDPConn()
		return up, err
	case u.Scheme == "tcp":
		err = up.createTCPConn()
		return up, err
	case u.Scheme == "syslog":
		err = up.createSyslog()
		return up, err
	}
	return nil, nil
}

func (self *UpStream) WriteData(logline *defines.Log) error {
	switch self.scheme {
	case "tcp":
		return self.writeJSON(logline)
	case "udp":
		return self.writeJSON(logline)
	case "syslog":
		tag := fmt.Sprintf("%s.%s", logline.Name, logline.Tag)
		remote, err := syslog.Dial("udp", self.addr, syslog.LOG_USER|syslog.LOG_INFO, tag)
		if err != nil {
			return err
		}
		_, err = io.WriteString(remote, logline.Data)
		return err
	default:
		return errors.New("Not support type")
	}
}

func (self *UpStream) createUDPConn() error {
	self.scheme = "udp"
	udpAddr, err := net.ResolveUDPAddr("udp", self.addr)
	if err != nil {
		logs.Info("Resolve", self.addr, "failed", err)
		return err
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		logs.Info("Connect backend failed", err)
		return err
	}
	self.conn = conn
	self.encoder = json.NewEncoder(conn)
	self.Close = conn.Close
	return nil
}

func (self *UpStream) createTCPConn() error {
	self.scheme = "tcp"
	tcpAddr, err := net.ResolveTCPAddr("tcp", self.addr)
	if err != nil {
		logs.Info("Resolve", self.addr, "failed", err)
		return err
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		logs.Debug("Connect backend failed", err)
		return err
	}
	self.conn = conn
	self.encoder = json.NewEncoder(conn)
	self.Close = conn.Close
	return nil
}

func (self *UpStream) createSyslog() error {
	self.scheme = "syslog"
	self.Close = func() error { return nil }
	return nil
}

func (self *UpStream) writeJSON(logline *defines.Log) error {
	self.buffer = append(self.buffer, logline)
	self.count += 1
	if self.count < g.Config.Lenz.Count {
		return nil
	}
	logs.Debug("Streamer buffer full, send to remote")
	return self.Flush()
}

func (self *UpStream) Tail() []*defines.Log {
	return self.buffer
}

func (self *UpStream) Flush() error {
	for i, log := range self.buffer {
		if err := self.encoder.Encode(log); err != nil {
			self.buffer = self.buffer[i:]
			return err
		}
	}
	self.buffer = []*defines.Log{}
	self.count = 0
	return nil
}
