package api

import (
	"github.com/HunanTV/eru-agent/g"
)

func Serve() {
	if g.Config.API.PubSub {
		go PubSubServe()
	}
	if g.Config.API.Http {
		go HTTPServe()
	}
}
