package defines

type DockerConfig struct {
	Endpoint string
	Registry string
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
}

type CleanerConfig struct {
	Interval int
	Dir      string
}

type LeviConfig struct {
	HostName string `yaml:"hostname"`
	PidFile  string

	Docker  DockerConfig
	Lenz    LenzConfig
	Metrics MetricsConfig
	Cleaner CleanerConfig
}
