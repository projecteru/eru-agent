package lenz

import (
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/HunanTV/eru-agent/defines"
	"github.com/HunanTV/eru-agent/logs"
	"github.com/HunanTV/eru-agent/utils"
)

type RouteStore interface {
	Get(id string) (*defines.Route, error)
	GetAll() ([]*defines.Route, error)
	Add(route *defines.Route) error
	Remove(id string) bool
}

type RouteManager struct {
	sync.Mutex
	persistor RouteStore
	attacher  *AttachManager
	routes    map[string]*defines.Route
	stdout    bool
}

func NewRouteManager(attacher *AttachManager, stdout bool) *RouteManager {
	return &RouteManager{attacher: attacher, routes: make(map[string]*defines.Route), stdout: stdout}
}

func (rm *RouteManager) Reload() error {
	newRoutes, err := rm.persistor.GetAll()
	if err != nil {
		return err
	}

	newRoutesMap := make(map[string]struct{})
	for _, newRoute := range newRoutes {
		newRoutesMap[newRoute.ID] = struct{}{}
		if route, ok := rm.routes[newRoute.ID]; ok {
			route.Source = newRoute.Source
			route.Target = newRoute.Target
			route.Backends = newRoute.Backends
			continue
		}
		rm.Add(newRoute)
	}

	for key, _ := range rm.routes {
		if _, ok := newRoutesMap[key]; ok || key == "lenz_default" {
			continue
		}
		rm.Remove(key)
	}
	return nil
}

func (rm *RouteManager) Load(persistor RouteStore) error {
	routes, err := persistor.GetAll()
	if err != nil {
		return err
	}
	for _, route := range routes {
		rm.Add(route)
	}
	rm.persistor = persistor
	return nil
}

func (rm *RouteManager) Get(id string) (*defines.Route, error) {
	rm.Lock()
	defer rm.Unlock()
	route, ok := rm.routes[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	return route, nil
}

func (rm *RouteManager) GetAll() ([]*defines.Route, error) {
	rm.Lock()
	defer rm.Unlock()
	routes := make([]*defines.Route, 0)
	for _, route := range rm.routes {
		routes = append(routes, route)
	}
	return routes, nil
}

func (rm *RouteManager) Add(route *defines.Route) error {
	rm.Lock()
	defer rm.Unlock()
	route.Closer = make(chan bool)
	rm.routes[route.ID] = route
	go func() {
		logstream := make(chan *defines.Log)
		defer close(logstream)
		go Streamer(route, logstream, rm.stdout)
		rm.attacher.Listen(route.Source, logstream, route.Closer)
	}()
	if rm.persistor != nil {
		if err := rm.persistor.Add(route); err != nil {
			logs.Info("Lenz Persistor:", err)
		}
	}
	return nil
}

func (rm *RouteManager) Remove(id string) bool {
	rm.Lock()
	defer rm.Unlock()
	route, ok := rm.routes[id]
	if ok && route.Closer != nil {
		route.Closer <- true
	}
	delete(rm.routes, id)
	if rm.persistor != nil {
		rm.persistor.Remove(id)
	}
	return ok
}

type RouteFileStore string

func (fs RouteFileStore) Filename(id string) string {
	return string(fs) + "/" + id + ".json"
}

func (fs RouteFileStore) Get(id string) (*defines.Route, error) {
	file, err := os.Open(fs.Filename(id))
	if err != nil {
		return nil, err
	}
	route := new(defines.Route)
	if err = utils.Unmarshal(file, route); err != nil {
		return nil, err
	}
	if route.ID == "" {
		route.ID = id
	}
	return route, nil
}

func (fs RouteFileStore) GetAll() ([]*defines.Route, error) {
	files, err := ioutil.ReadDir(string(fs))
	if err != nil {
		return nil, err
	}
	var routes []*defines.Route
	for _, file := range files {
		fileparts := strings.Split(file.Name(), ".")
		if len(fileparts) > 1 && fileparts[1] == "json" {
			route, err := fs.Get(fileparts[0])
			if err == nil {
				routes = append(routes, route)
			}
			route.LoadBackends()
		}
	}
	return routes, nil
}

func (fs RouteFileStore) Add(route *defines.Route) error {
	return ioutil.WriteFile(fs.Filename(route.ID), utils.Marshal(route), 0644)
}

func (fs RouteFileStore) Remove(id string) bool {
	if _, err := os.Stat(fs.Filename(id)); err == nil {
		if err := os.Remove(fs.Filename(id)); err != nil {
			return true
		}
	}
	return false
}
