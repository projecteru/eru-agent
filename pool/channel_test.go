package pool

import (
	"log"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"
)

var (
	InitialCap = 5
	MaximumCap = 30
	network    = "tcp"
	address    = "127.0.0.1:7777"
)

func init() {
	// used for factory function
	go simpleTCPServer()
	time.Sleep(time.Millisecond * 300) // wait until tcp server has been settled

	rand.Seed(time.Now().UTC().UnixNano())
}

func TestNew(t *testing.T) {
	_, err := NewChannelPool(InitialCap, MaximumCap)
	if err != nil {
		t.Errorf("New error: %s", err)
	}
}

func TestPool_Get_Impl(t *testing.T) {
	p, _ := NewChannelPool(InitialCap, MaximumCap)
	defer p.Close()

	_, err := p.Get(address, network)
	if err != nil {
		t.Errorf("Get error: %s", err)
	}
}

func TestPool_Get(t *testing.T) {
	p, _ := NewChannelPool(InitialCap, MaximumCap)
	defer p.Close()

	_, err := p.Get(address, network)
	if err != nil {
		t.Errorf("Get error: %s", err)
	}

	if p.Len() != (InitialCap - 1) {
		t.Errorf("Get error. Expecting %d, got %d", (InitialCap - 1), p.Len())
	}

	// get them all
	var wg sync.WaitGroup
	for i := 0; i < (InitialCap - 1); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := p.Get(address, network)
			if err != nil {
				t.Errorf("Get error: %s", err)
			}
		}()
	}
	wg.Wait()

	if p.Len() != 0 {
		t.Errorf("Get error. Expecting %d, got %d", (InitialCap - 1), p.Len())
	}

	_, err = p.Get(address, network)
	if err != nil {
		t.Errorf("Get error: %s", err)
	}
}

func TestPool_Put(t *testing.T) {
	p, err := NewChannelPool(0, 30)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// get/create from the pool
	conns := make([]net.Conn, MaximumCap)
	for i := 0; i < MaximumCap; i++ {
		conn, _ := p.Get(address, network)
		conns[i] = conn
	}

	// now put them all back
	for _, conn := range conns {
		p.Put(conn, address, network)
	}

	if p.Len() != MaximumCap {
		t.Errorf("Put error len. Expecting %d, got %d", 1, p.Len())
	}

	conn, _ := p.Get(address, network)
	p.Close() // close pool

	p.Put(conn, address, network)
	if p.Len() != 0 {
		t.Errorf("Put error. Closed pool shouldn't allow to put connections.")
	}
}

func TestPoolConcurrent(t *testing.T) {
	p, _ := NewChannelPool(5, 30)
	pipe := make(chan net.Conn, 0)

	go func() {
		p.Close()
	}()

	for i := 0; i < MaximumCap; i++ {
		go func() {
			conn, _ := p.Get(address, network)
			pipe <- conn
		}()

		go func() {
			conn := <-pipe
			if conn == nil {
				return
			}
			p.Put(conn, address, network)
			conn.Close()
		}()
	}
}

func TestPoolWriteRead(t *testing.T) {
	p, _ := NewChannelPool(0, 30)

	conn, _ := p.Get(address, network)

	msg := "hello"
	_, err := conn.Write([]byte(msg))
	if err != nil {
		t.Error(err)
	}
}

func TestPoolConcurrent2(t *testing.T) {
	p, _ := NewChannelPool(0, 30)

	var wg sync.WaitGroup

	go func() {
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				conn, _ := p.Get(address, network)
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
				conn.Close()
				wg.Done()
			}(i)
		}
	}()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			conn, _ := p.Get(address, network)
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
			conn.Close()
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func simpleTCPServer() {
	l, err := net.Listen(network, address)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			buffer := make([]byte, 256)
			conn.Read(buffer)
		}()
	}
}
