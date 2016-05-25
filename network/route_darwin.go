package network

import (
	log "github.com/Sirupsen/logrus"
	"github.com/projecteru/eru-agent/g"
	"golang.org/x/net/context"
)

func SetDefaultRoute(cid, gateway string) bool {
	ctx := context.Background()
	_, err := g.Docker.ContainerInspect(ctx, cid)
	if err != nil {
		log.Errorf("VLanSetter inspect docker failed %s", err)
		return false
	}
	log.Infof("Set default route success %s %s", cid, gateway)
	return true
}

func AddRoute(cid, CIDR string, ifc string) bool {
	ctx := context.Background()
	_, err := g.Docker.ContainerInspect(ctx, cid)
	if err != nil {
		log.Errorf("VLanSetter inspect docker failed %s", err)
		return false
	}
	log.Infof("Add route success %s %s %s", cid, CIDR, ifc)
	return true
}
