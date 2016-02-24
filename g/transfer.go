package g

import (
	"github.com/projecteru/eru-agent/logs"
	"github.com/projecteru/eru-agent/utils"
)

var Transfers *utils.HashBackends

func InitTransfers() {
	Transfers = utils.NewHashBackends(Config.Metrics.Transfers)
	logs.Info("Transfers initiated")
}
