package defines

type DockerConfig struct {
	Endpoint string
}

type EruConfig struct {
	Endpoint string
}

type LenzConfig struct {
	Routes   string
	Forwards []string
	Stdout   bool
	Count    int
}

type MetricsConfig struct {
	Step      int64
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
	Calico   string
}

type APIConfig struct {
	Addr string
}

type LimitConfig struct {
	Memory uint64
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
}
