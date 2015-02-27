# Eru-Agent

## Features

Agent of Project Eru, for those reasons:

1. Check containers status, if crashed, report to eru core.
2. Get containers metrics, send to [InfluxDB](http://influxdb.com/).
3. Clean containers' log files in disk.
4. Forwarding containers' log to remote collectors like syslog.

## Run

Agent has a configure file named `agent.yaml`, you can execute agent like:

    agent -c agent.yaml [-DEBUG]

