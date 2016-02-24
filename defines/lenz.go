package defines

import "github.com/projecteru/eru-agent/utils"

type AttachEvent struct {
	Type string
	App  *Meta
}

type Log struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	EntryPoint string `json:"entrypoint"`
	Ident      string `json:"ident"`
	Data       string `json:"data"`
	Tag        string `json:"tag"`
	Count      int64  `json:"count"`
	Datetime   string `json:"datetime"`
}

type Route struct {
	ID       string  `json:"id"`
	Source   *Source `json:"source,omitempty"`
	Target   *Target `json:"target"`
	Backends *utils.HashBackends
	Closer   chan bool
	Done     chan struct{}
}

func (s *Route) LoadBackends() {
	s.Backends = utils.NewHashBackends(s.Target.Addrs)
}

type Source struct {
	ID     string   `json:"id,omitempty"`
	Name   string   `json:"name,omitempty"`
	Filter string   `json:"filter,omitempty"`
	Types  []string `json:"types,omitempty"`
}

func (s *Source) All() bool {
	return s.ID == "" && s.Name == "" && s.Filter == ""
}

type Target struct {
	Addrs     []string `json:"addrs"`
	AppendTag string   `json:"append_tag,omitempty"`
}
