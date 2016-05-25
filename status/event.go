package status

import (
	"sync"

	log "github.com/Sirupsen/logrus"
	eventtypes "github.com/docker/engine-api/types/events"
)

type EventProcesser func(event eventtypes.Message, err error) error

type EventHandler struct {
	sync.Mutex
	handlers map[string]func(eventtypes.Message)
}

func NewEventHandler() *EventHandler {
	return &EventHandler{handlers: make(map[string]func(eventtypes.Message))}
}

func (self *EventHandler) Handle(action string, h func(eventtypes.Message)) {
	self.Lock()
	self.handlers[action] = h
	self.Unlock()
}

func (self *EventHandler) Watch(c <-chan eventtypes.Message) {
	for e := range c {
		log.Debugf("event handler: received event: %v", e)
		self.Lock()
		h, exists := self.handlers[e.Action]
		self.Unlock()
		if !exists {
			continue
		}
		go h(e)
	}
}
