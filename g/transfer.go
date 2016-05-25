package g

import (
	"github.com/projecteru/eru-agent/utils"

	log "github.com/Sirupsen/logrus"
)

var Transfers *utils.HashBackends

func InitTransfers() {
	Transfers = utils.NewHashBackends(Config.Metrics.Transfers)
	log.Info("Transfers initiated")
}
