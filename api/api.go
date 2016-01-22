package api

import (
	"github.com/projecteru/eru-agent/g"
)

func Serve() {
	if g.Config.API.Addr != "" {
		go HTTPServe()
	}
}
