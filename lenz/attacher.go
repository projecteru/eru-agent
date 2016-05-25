package lenz

import (
	"bufio"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/projecteru/eru-agent/common"
	"github.com/projecteru/eru-agent/defines"
	"github.com/projecteru/eru-agent/g"

	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
)

type AttachManager struct {
	sync.Mutex
	attached map[string]*LogPump
	channels map[chan *defines.AttachEvent]struct{}
}

func NewAttachManager() *AttachManager {
	m := &AttachManager{
		attached: make(map[string]*LogPump),
		channels: make(map[chan *defines.AttachEvent]struct{}),
	}
	return m
}

func (m *AttachManager) Attached(id string) bool {
	_, ok := m.attached[id]
	return ok
}

func (m *AttachManager) Attach(app *defines.Meta) {
	//TODO Not Thread Safe
	if m.Attached(app.ID) {
		return
	}

	outr, outw := io.Pipe()
	go func() {
		ctx := context.Background()
		options := types.ContainerAttachOptions{
			Stream: true,
			Stdin:  false,
			Stdout: true,
			Stderr: true,
		}
		resp, err := g.Docker.ContainerAttach(ctx, app.ID, options)
		if err != nil {
			log.Errorf("Lenz Attach %s failed %s", app.ID[:12], err)
			return
		}
		defer resp.Close()
		_, err = io.Copy(outw, resp.Reader)
		outw.Close()
		log.Debugf("Lenz Attach %s finished", app.ID[:12])
		if err != nil {
			log.Errorf("Lenz Attach get stream failed %s", err)
		}
		m.send(&defines.AttachEvent{Type: "detach", App: app})
		m.Lock()
		defer m.Unlock()
		delete(m.attached, app.ID)
	}()
	m.Lock()
	m.attached[app.ID] = NewLogPump(outr, app)
	m.Unlock()
	m.send(&defines.AttachEvent{Type: "attach", App: app})
	log.Debugf("Lenz Attach %s success", app.ID[:12])
}

func (m *AttachManager) send(event *defines.AttachEvent) {
	m.Lock()
	defer m.Unlock()
	for ch, _ := range m.channels {
		// TODO: log err after timeout and continue
		ch <- event
	}
}

func (m *AttachManager) addListener(ch chan *defines.AttachEvent) {
	m.Lock()
	defer m.Unlock()
	m.channels[ch] = struct{}{}
	go func() {
		for _, pump := range m.attached {
			ch <- &defines.AttachEvent{Type: "attach", App: pump.app}
		}
	}()
}

func (m *AttachManager) removeListener(ch chan *defines.AttachEvent) {
	m.Lock()
	defer m.Unlock()
	delete(m.channels, ch)
}

func (m *AttachManager) Get(id string) *LogPump {
	m.Lock()
	defer m.Unlock()
	return m.attached[id]
}

func (m *AttachManager) Listen(source *defines.Source, logstream chan *defines.Log, closer <-chan bool) {
	if source == nil {
		source = new(defines.Source)
	}
	events := make(chan *defines.AttachEvent)
	m.addListener(events)
	defer m.removeListener(events)
	for {
		select {
		case event := <-events:
			if event.Type == "attach" && (source.All() ||
				(source.ID != "" && strings.HasPrefix(event.App.ID, source.ID)) ||
				(source.Name != "" && event.App.Name == source.Name) ||
				(source.Filter != "" && strings.Contains(event.App.Name, source.Filter))) {
				pump := m.Get(event.App.ID)
				pump.AddListener(logstream)
				defer func() {
					if pump != nil {
						pump.RemoveListener(logstream)
					}
				}()
			} else if source.ID != "" && event.Type == "detach" &&
				strings.HasPrefix(event.App.ID, source.ID) {
				return
			}
		case <-closer:
			return
		}
	}
}

type LogPump struct {
	sync.Mutex
	app      *defines.Meta
	channels map[chan *defines.Log]struct{}
}

func NewLogPump(stream io.Reader, app *defines.Meta) *LogPump {
	obj := &LogPump{
		app:      app,
		channels: make(map[chan *defines.Log]struct{}),
	}
	pump := func(typ string, source io.Reader) {
		buf := bufio.NewReader(source)
		for {
			data, err := buf.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					log.Errorf("Lenz Pump: %s %s %s", app.ID, typ, err)
				}
				return
			}
			obj.send(&defines.Log{
				Data:       strings.TrimSuffix(string(data), "\n"),
				ID:         app.ID,
				Name:       app.Name,
				EntryPoint: app.EntryPoint,
				Ident:      app.Ident,
				Type:       typ,
				Datetime:   time.Now().Format(common.DATETIME_FORMAT),
			})
		}
	}
	go pump("stream", stream)
	return obj
}

func (o *LogPump) send(log *defines.Log) {
	o.Lock()
	defer o.Unlock()
	for ch, _ := range o.channels {
		// TODO: log err after timeout and continue
		ch <- log
	}
}

func (o *LogPump) AddListener(ch chan *defines.Log) {
	o.Lock()
	defer o.Unlock()
	o.channels[ch] = struct{}{}
}

func (o *LogPump) RemoveListener(ch chan *defines.Log) {
	o.Lock()
	defer o.Unlock()
	delete(o.channels, ch)
}
