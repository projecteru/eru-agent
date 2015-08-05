package defines

type DockerConfig struct {
	Endpoint string
	Ca       string
	Key      string
	Cert     string
	Health   int
}

type EruConfig struct {
	Endpoint string
}

type LenzConfig struct {
	Routes   string
	Forwards []string
	Stdout   bool
}

type MetricsConfig struct {
	Step      int64
	Timeout   int64
	Transfers []string
}

type RedisConfig struct {
	Host string
	Port int
	Min  int
	Max  int
}

type VLanConfig struct {
	Physical []string
}

type APIConfig struct {
	Http   bool
	PubSub bool
	Addr   string
}

type LimitConfig struct {
	Memory uint64
}

type CalicoConfig struct {
	NodeIP    string `yaml:"node_ip"`
	NodeImage string `yaml:"node_image"`
}

type AgentConfig struct {
	HostName string `yaml:"hostname"`
	PidFile  string

	Docker  DockerConfig
	Eru     EruConfig
	Lenz    LenzConfig
	Metrics MetricsConfig
	VLan    VLanConfig
	Redis   RedisConfig
	API     APIConfig
	Limit   LimitConfig
	Calico  CalicoConfig
}
