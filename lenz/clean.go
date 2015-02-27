package lenz

import (
	"os"
	"path/filepath"
	"time"

	"../defines"
	"../logs"
)

type Cleaner struct {
	interval int
	path     string
	stop     chan bool
}

func NewCleaner(config defines.CleanerConfig) *Cleaner {
	c := &Cleaner{
		interval: config.Interval,
		stop:     make(chan bool),
	}
	c.path = filepath.Join(config.Dir, "*", "*-json.log")
	c.doClean()
	return c
}

func (self *Cleaner) doClean() {
	fs, err := filepath.Glob(self.path)
	if err != nil {
		logs.Debug("Cleaner", err)
		return
	}
	for _, f := range fs {
		logs.Debug("Cleaner Truncate", f)
		os.Truncate(f, 0)
	}
}

func (self *Cleaner) Clean() {
	defer close(self.stop)
	for {
		select {
		case <-time.After(time.Second * time.Duration(self.interval)):
			logs.Debug("Cleaner Get Log Files")
			self.doClean()
		case <-self.stop:
			logs.Info("Cleaner Stop")
			return
		}
	}
}

func (self *Cleaner) Stop() {
	self.stop <- true
}
