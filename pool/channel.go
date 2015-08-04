package pool

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/HunanTV/eru-agent/logs"
)

type channelPool struct {
	sync.Mutex
	conns   map[string]chan net.Conn
	initCap int
	maxCap  int
}

func formatAddrProtocol(addr, proto string) string {
	return fmt.Sprintf("%s://%s", proto, addr)
}

func createTCPConn(addr string) (net.Conn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		logs.Debug("Resolve tcp failed", err)
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		logs.Debug("Connection backend failed", err)
		return nil, err
	}
	return conn, nil
}

func createUDPConn(addr string) (net.Conn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		logs.Debug("Resolve udp failed", err)
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		logs.Debug("Connection backend failed", err)
		return nil, err
	}
	return conn, nil
}

func createConn(addr, proto string) (net.Conn, error) {
	var (
		conn net.Conn
		err  error
	)
	if proto == "udp" || proto == "UDP" {
		conn, err = createUDPConn(addr)
	} else if proto == "tcp" || proto == "TCP" {
		conn, err = createTCPConn(addr)
	} else {
		return nil, errors.New("only support tcp/udp")
	}
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func NewChannelPool(initCap, maxCap int) (Pool, error) {
	if initCap < 0 || maxCap <= 0 || initCap > maxCap {
		return nil, errors.New("invalid capacity settings")
	}

	c := &channelPool{
		conns:   map[string]chan net.Conn{},
		initCap: initCap,
		maxCap:  maxCap,
	}
	return c, nil
}

func (c *channelPool) initAddrProtoConn(addr, proto string) (chan net.Conn, error) {
	key := formatAddrProtocol(addr, proto)
	connChannel := make(chan net.Conn, c.maxCap)

	for i := 0; i < c.initCap; i++ {
		conn, err := createConn(addr, proto)
		if err != nil {
			return nil, err
		}
		connChannel <- conn
	}
	c.conns[key] = connChannel
	return connChannel, nil
}

func (c *channelPool) getConns(addr, proto string, needLock bool) (chan net.Conn, error) {
	if needLock {
		c.Lock()
		defer c.Unlock()
	}

	if c.conns == nil {
		return nil, errors.New("Closed")
	}

	key := formatAddrProtocol(addr, proto)
	r, ok := c.conns[key]
	if ok {
		return r, nil
	}

	connChannel, err := c.initAddrProtoConn(addr, proto)
	if err != nil {
		return nil, err
	}
	return connChannel, nil
}

func (c *channelPool) Get(addr, proto string) (net.Conn, error) {
	conns, err := c.getConns(addr, proto, true)
	if err != nil {
		return nil, err
	}

	select {
	case conn := <-conns:
		if conn == nil {
			return nil, ErrClosed
		}
		return conn, nil
	default:
		return createConn(addr, proto)
	}
}

func (c *channelPool) Put(conn net.Conn, addr, proto string) error {
	if conn == nil {
		return errors.New("connection is nil. rejecting")
	}

	c.Lock()
	defer c.Unlock()

	conns, err := c.getConns(addr, proto, false)
	if conns == nil || err != nil {
		return conn.Close()
	}

	select {
	case conns <- conn:
		return nil
	default:
		return conn.Close()
	}
}

func (c *channelPool) Close() {
	c.Lock()
	defer c.Unlock()

	conns := c.conns
	c.conns = nil

	if conns == nil {
		return
	}

	for _, connChannel := range conns {
		close(connChannel)
		for conn := range connChannel {
			conn.Close()
		}
	}
}

func (c *channelPool) Len() int {
	length := 0
	for _, connChannel := range c.conns {
		length = length + len(connChannel)
	}
	return length
}

var ConnPool Pool

func InitPool() {
	var err error
	ConnPool, err = NewChannelPool(5, 30)
	if err != nil {
		logs.Assert(err, "Pool not initialized")
	}
}

func ClosePool() {
	ConnPool.Close()
}
