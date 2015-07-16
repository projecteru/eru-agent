package g

import (
	"sync"

	"github.com/HunanTV/eru-agent/defines"
)

var lock sync.RWMutex
var Apps map[string]*defines.App = map[string]*defines.App{}

func AddApp(app *defines.App) {
	lock.Lock()
	defer lock.Unlock()
	if _, ok := Apps[app.ID]; ok {
		// safe add
		return
	}
	Apps[app.ID] = app
}

func RemoveApp(ID string) {
	lock.Lock()
	defer lock.Unlock()
	if _, ok := Apps[ID]; !ok {
		return
	}
	delete(Apps, ID)
}

func VaildApp(ID string) bool {
	lock.RLock()
	defer lock.RUnlock()
	_, ok := Apps[ID]
	return ok
}
