package lenz

import (
	"bufio"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/HunanTV/eru-agent/common"
	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/g"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/fsouza/go-dockerclient"
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
	// Not Thread Safe
	if m.Attached(app.ID) {
		return
	}
	success := make(chan struct{})
	failure := make(chan error)
	outrd, outwr := io.Pipe()
	errrd, errwr := io.Pipe()
	go func() {
		err := g.Docker.AttachToContainer(docker.AttachToContainerOptions{
			Container:    app.ID,
			OutputStream: outwr,
			ErrorStream:  errwr,
			Stdin:        false,
			Stdout:       true,
			Stderr:       true,
			Stream:       true,
			Success:      success,
		})
		outwr.Close()
		errwr.Close()
		logs.Debug("Lenz Attach", app.ID[:12], "finished")
		if err != nil {
			close(success)
			failure <- err
		}
		m.send(&defines.AttachEvent{Type: "detach", App: app})
		m.Lock()
		defer m.Unlock()
		delete(m.attached, app.ID)
	}()
	_, ok := <-success
	if ok {
		m.Lock()
		m.attached[app.ID] = NewLogPump(outrd, errrd, app)
		m.Unlock()
		success <- struct{}{}
		m.send(&defines.AttachEvent{Type: "attach", App: app})
		logs.Debug("Lenz Attach", app.ID[:12], "success")
		return
	}
	logs.Debug("Lenz Attach", app.ID, "failure:", <-failure)
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

func NewLogPump(stdout, stderr io.Reader, app *defines.Meta) *LogPump {
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
					logs.Debug("Lenz Pump:", app.ID, typ, err)
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
	go pump("stdout", stdout)
	go pump("stderr", stderr)
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
