package metrics

import (
	"math"
	"net/rpc"
	"sync"
	"time"

	"github.com/HunanTV/eru-agent/logs"

	"github.com/toolkits/net"
)

type SingleConnRpcClient struct {
	sync.Mutex
	rpcClient *rpc.Client
	RpcServer string
	Timeout   time.Duration
}

func (this *SingleConnRpcClient) close() {
	if this.rpcClient != nil {
		this.rpcClient.Close()
		this.rpcClient = nil
	}
}

func (this *SingleConnRpcClient) insureConn() error {
	if this.rpcClient != nil {
		return nil
	}

	var err error
	var retry int = 1

	for {
		if this.rpcClient != nil {
			return nil
		}

		this.rpcClient, err = net.JsonRpcClient("tcp", this.RpcServer, this.Timeout)
		if err == nil {
			return nil
		}

		logs.Info("Metrics rpc dial fail", this.RpcServer, err)
		if retry > 5 {
			return err
		}

		time.Sleep(time.Duration(math.Pow(2.0, float64(retry))) * time.Second)
		retry++
	}
	return nil
}

func (this *SingleConnRpcClient) Call(method string, args interface{}, reply interface{}) error {

	this.Lock()
	defer this.Unlock()

	if err := this.insureConn(); err != nil {
		return err
	}

	timeout := time.Duration(50 * time.Second)
	done := make(chan error)

	go func() {
		err := this.rpcClient.Call(method, args, reply)
		done <- err
	}()

	select {
	case <-time.After(timeout):
		logs.Info("Metrics rpc call timeout", this.rpcClient, this.RpcServer)
		this.close()
	case err := <-done:
		if err != nil {
			this.close()
			return err
		}
	}

	return nil
}
