package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type JSON map[string]interface{}

func JSONWrapper(f func(*Request) (int, interface{})) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		code, r := NewRequest(req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(f(r))
	}
}

func Atoi(s string, def int) int {
	if r, err := strconv.Atoi(s); err != nil {
		return def
	} else {
		return r
	}
}
