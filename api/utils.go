package api

import (
	"encoding/json"
	"net/http"
)

type JSON map[string]interface{}

func JSONWrapper(f func(*Request) interface{}) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		r := NewRequest(req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(f(r))
	}
}
