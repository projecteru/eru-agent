package defines

import (
	"github.com/HunanTV/eru-agent/logs"
	"github.com/HunanTV/eru-agent/utils"
)

type App struct {
	ID         string
	Name       string
	EntryPoint string
	Ident      string
}

func NewApp(ID, containerName string) *App {
	name, entrypoint, ident := utils.GetAppInfo(containerName)
	if name == "" {
		// ignore
		logs.Info("Container name invald", containerName)
		return nil
	}
	logs.Debug("Container", name, entrypoint, ident)
	return &App{ID, name, entrypoint, ident}
}
