package defines

import (
	"fmt"

	"github.com/docker/engine-api/client"
	"github.com/projecteru/eru-agent/common"
)

func NewDocker(endpoint string) (*client.Client, error) {
	defaultHeaders := map[string]string{"User-Agent": fmt.Sprintf("eru-agent-%s", common.VERSION)}
	return client.NewClient(endpoint, common.DOCKER_CLI_VERSION, nil, defaultHeaders)
}
