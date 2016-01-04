package api

import (
	"github.com/HunanTV/eru-agent/g"
)

func Serve() {
	if g.Config.API.Addr != "" {
		go HTTPServe()
	}
}
