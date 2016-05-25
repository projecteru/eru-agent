package status

import (
	"encoding/json"
	"io"

	log "github.com/Sirupsen/logrus"
	eventtypes "github.com/docker/engine-api/types/events"

	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
	"golang.org/x/net/context"
)

func decodeEvents(input io.Reader, ep EventProcesser) error {
	dec := json.NewDecoder(input)
	for {
		var event eventtypes.Message
		err := dec.Decode(&event)
		if err != nil && err == io.EOF {
			break
		}

		if procErr := ep(event, err); procErr != nil {
			return procErr
		}
	}
	return nil
}

func MonitorContainerEvents(errChan chan<- error, c chan eventtypes.Message) {
	ctx := context.Background()
	f := filters.NewArgs()
	f.Add("type", "container")
	options := types.EventsOptions{
		Filters: f,
	}
	resBody, err := cli.Events(ctx, options)
	// Whether we successfully subscribed to events or not, we can now
	// unblock the main goroutine.
	if err != nil {
		errChan <- err
		return
	}
	log.Info("Status watch start")
	defer resBody.Close()

	decodeEvents(resBody, func(event eventtypes.Message, err error) error {
		if err != nil {
			errChan <- err
			return nil
		}
		c <- event
		return nil
	})
}
