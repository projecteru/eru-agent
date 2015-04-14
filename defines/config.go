package defines

type DockerConfig struct {
	Endpoint string
	Ca       string
	Key      string
	Cert     string
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
	ReportInterval int
	Host           string
	Username       string
	Password       string
	Database       string
	Retention      string
	Precision      string
}

type CleanerConfig struct {
	Interval int
	Dir      string
}

type RedisConfig struct {
	Host string
	Port int
	Min  int
	Max  int
}

type MacvlanConfig struct {
	Physical []string
}

type AgentConfig struct {
	HostName string `yaml:"hostname"`
	PidFile  string

	Docker  DockerConfig
	Eru     EruConfig
	Lenz    LenzConfig
	Metrics MetricsConfig
	Macvlan MacvlanConfig
	Cleaner CleanerConfig
	Redis   RedisConfig
}
