package main

import (
	"net/http"
)

type VersionHandler struct {
	Version string
}

func (h VersionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(h.Version + "\n"))
}
