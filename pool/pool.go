package pool

import (
	"errors"
	"net"
)

var (
	ErrClosed = errors.New("pool is closed")
)

type Pool interface {
	Get(addr, proto string) (net.Conn, error)
	Put(c net.Conn, addr, proto string) error
	Close()
	Len() int
}
