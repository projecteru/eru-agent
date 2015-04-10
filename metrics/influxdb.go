package metrics

import (
	"net/url"
	"time"

	"../defines"
	"../logs"
	"github.com/influxdb/influxdb/client"
)

type InfluxDBClient struct {
	hostname  string
	database  string
	retention string
	precision string
	client    *client.Client
	points    []client.Point
}

func NewInfluxDBClient(hostname string, config defines.MetricsConfig) *InfluxDBClient {
	host, _ := url.Parse(config.Host)
	c := client.Config{
		URL:      *host,
		Username: config.Username,
		Password: config.Password,
	}

	i, err := client.NewClient(c)
	if err != nil {
		logs.Assert(err, "InfluxDB")
	}
	return &InfluxDBClient{
		hostname, config.Database,
		config.Retention, config.Precision,
		i, []client.Point{},
	}
}

func (self *InfluxDBClient) GenSeries(ID string, metric *MetricData) {
	point := client.Point{
		Name: metric.app.Name,
		Tags: map[string]string{
			"hostname":   self.hostname,
			"ID":         ID,
			"entrypoint": metric.app.EntryPoint,
			"ident":      metric.app.Ident,
		},
		Fields: map[string]interface{}{
			"cpu_usage":       metric.cpu_usage,
			"cpu_system":      metric.cpu_system,
			"cpu_user":        metric.cpu_user,
			"mem_usage":       metric.mem_usage,
			"mem_max_usage":   metric.mem_max_usage,
			"mem_rss":         metric.mem_rss,
			"cpu_usage_rate":  metric.cpu_user_rate,
			"cpu_system_rate": metric.cpu_system_rate,
			"cpu_user_rate":   metric.cpu_user_rate,
		},
		Timestamp: time.Now(),
		Precision: self.precision,
	}
	for key, data := range metric.network {
		point.Fields[key] = data
	}
	for key, data := range metric.network_rate {
		point.Fields[key] = data
	}
	self.points = append(self.points, point)
}

func (self *InfluxDBClient) Send() {
	bps := client.BatchPoints{
		Points:          self.points,
		Database:        self.database,
		RetentionPolicy: self.retention,
	}
	if _, err := self.client.Write(bps); err != nil {
		logs.Info("Write to InfluxDB Failed", err)
	}
	self.points = []client.Point{}
}
