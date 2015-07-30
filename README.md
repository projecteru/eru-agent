Eru Agent
=========

## Features

Agent of Project Eru, for those reasons:

1. Check containers status, if crashed, report to eru core.
2. Get containers metrics, send to [Open-Falcon](https://github.com/open-falcon).
3. Forwarding containers' log to remote collectors like syslog.

## Run

Agent has a configure file named `agent.yaml`, you can execute agent like:

    agent -c agent.yaml [-DEBUG]

## APIs

### PubSub

1. Add Vlan(s)

```
PUBLISH eru:agent:127.0.0.1:vlan aaa|889c8eb8d6d45aa1cfbe36ebe30933bea22be8c890173b425b3378a966e1bfe5|1:10.1.1.1|2:10.2.2.2
```

It will add 2 MacVlan devices in container, named vnbe1 and vnbe2, bound 10.1.1.1 and 10.2.2.2.

2. Add EruApp

```
PUBLISH eru:agent:127.0.0.1:watcher '+|889c8eb8d6d45aa1cfbe36ebe30933bea22be8c890173b425b3378a966e1bfe5|{"a": 1, "c": "123", "b": 2}'
```

3. Set Default Route

```
PUBLISH eru:agent:127.0.0.1:route 889c8eb8d6d45aa1cfbe36ebe30933bea22be8c890173b425b3378a966e1bfe5|172.42.1.1
```

### HTTP

Coming soon...
