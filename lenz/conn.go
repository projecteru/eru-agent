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
	"github.com/HunanTV/eru-agent/logs"
)

type UpStream struct {
	addr   string
	scheme string
	tcplog *net.TCPConn
	udplog *net.UDPConn
	Close  func() error
}

func NewUpStream(addr string) (up *UpStream, err error) {
	u, err := url.Parse(addr)
	if err != nil {
		logs.Info("Parse upstream addr failed", err)
		return nil, err
	}
	up = &UpStream{addr: u.Host}
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
		return writeJSON(self.tcplog, logline)
	case "udp":
		return writeJSON(self.udplog, logline)
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
	self.udplog = conn
	self.Close = self.udplog.Close
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
	self.tcplog = conn
	self.Close = self.tcplog.Close
	return nil
}

func (self *UpStream) createSyslog() error {
	self.scheme = "syslog"
	self.Close = func() error { return nil }
	return nil
}

func writeJSON(w io.Writer, logline *defines.Log) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(logline)
}
