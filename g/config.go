package g

import (
	"flag"
	"io/ioutil"
	"os"

	"github.com/projecteru/eru-agent/common"
	"github.com/projecteru/eru-agent/defines"
	"gopkg.in/yaml.v2"

	log "github.com/Sirupsen/logrus"
)

var Config = defines.AgentConfig{}

func LoadConfig() {
	var configPath string
	var version bool
	var debug bool
	flag.BoolVar(&debug, "DEBUG", false, "enable debug")
	flag.StringVar(&configPath, "c", "agent.yaml", "config file")
	flag.BoolVar(&version, "v", false, "show version")
	flag.Parse()
	if debug {
		log.SetLevel(log.DebugLevel)
	}
	if version {
		log.Infof("Version %s", common.VERSION)
		os.Exit(0)
	}
	load(configPath)
}

func load(configPath string) {
	if _, err := os.Stat(configPath); err != nil {
		log.Panicf("Config file invaild %s", err)
	}

	b, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Panicf("Read config file failed %s", err)
	}

	if err := yaml.Unmarshal(b, &Config); err != nil {
		log.Panicf("Load config file failed %s", err)
	}

	if Config.HostName, err = os.Hostname(); err != nil {
		log.Panicf("Load hostname failed %s", err)
	}
	log.Debugf("Configure: %s", Config)
}
