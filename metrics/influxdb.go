package metrics

import (
	"net/http"

	"../defines"
	"../logs"
	"github.com/influxdb/influxdb/client"
)

type InfluxDBClient struct {
	hostname string
	client   *client.Client
	series   []*client.Series
}

var influxdb_columns []string = []string{"host", "ID", "entrypoint", "ident", "metric", "value"}

func NewInfluxDBClient(hostname string, config defines.MetricsConfig) *InfluxDBClient {
	c := &client.ClientConfig{
		Host:       config.Host,
		Username:   config.Username,
		Password:   config.Password,
		Database:   config.Database,
		HttpClient: http.DefaultClient,
		IsSecure:   false,
		IsUDP:      false,
	}
	i, err := client.New(c)
	if err != nil {
		logs.Assert(err, "InfluxDB")
	}
	return &InfluxDBClient{hostname, i, []*client.Series{}}
}

func (self *InfluxDBClient) GenSeries(ID string, metric *MetricData) {
	points := [][]interface{}{
		{self.hostname, ID, metric.app.EntryPoint, metric.app.Ident, "cpu_usage", metric.cpu_usage},
		{self.hostname, ID, metric.app.EntryPoint, metric.app.Ident, "cpu_system", metric.cpu_system},
		{self.hostname, ID, metric.app.EntryPoint, metric.app.Ident, "cpu_user", metric.cpu_user},
		{self.hostname, ID, metric.app.EntryPoint, metric.app.Ident, "mem_usage", metric.mem_usage},
		{self.hostname, ID, metric.app.EntryPoint, metric.app.Ident, "mem_rss", metric.mem_rss},
		{self.hostname, ID, metric.app.EntryPoint, metric.app.Ident, "net_recv", metric.net_inbytes},
		{self.hostname, ID, metric.app.EntryPoint, metric.app.Ident, "net_send", metric.net_outbytes},
		{self.hostname, ID, metric.app.EntryPoint, metric.app.Ident, "net_recv_err", metric.net_inerrs},
		{self.hostname, ID, metric.app.EntryPoint, metric.app.Ident, "net_send_err", metric.net_outerrs},
	}
	series := &client.Series{
		Name:    metric.app.Name,
		Columns: influxdb_columns,
		Points:  points,
	}
	self.series = append(self.series, series)
}

func (self *InfluxDBClient) Send() {
	if err := self.client.WriteSeries(self.series); err != nil {
		logs.Info("Write to InfluxDB Failed", err)
	}
	self.series = []*client.Series{}
}
