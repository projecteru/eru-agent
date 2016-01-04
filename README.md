Eru Agent
=========

## Features

Agent of Project Eru, for those reasons:

1. Check containers status, if crashed, report to eru core.
2. Get containers metrics, send to [Open-Falcon](https://github.com/open-falcon).
3. Forwarding containers' log to remote collectors like syslog.
4. Support macvlan and calico SDN.

## Run

Agent has a configure file named `agent.yaml`, you can execute agent like:

    agent -c agent.yaml [-DEBUG]

## APIs

### HTTP

Coming soon...
