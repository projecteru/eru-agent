package g

import (
	"github.com/CMGS/consistent"
	"github.com/HunanTV/eru-agent/logs"
)

var Transfers *consistent.Consistent

func InitTransfers() {
	Transfers = consistent.New()
	for _, transfer := range Config.Metrics.Transfers {
		Transfers.Add(transfer)
	}
	logs.Info("Transfers initiated")
}
