package main

import (
	"fmt"
	"log"
	"net/http"
)

func respondErr(w http.ResponseWriter, code int, msg string, args ...interface{}) {
	w.WriteHeader(code)
	msg = fmt.Sprintf("%d: %s", code, fmt.Sprintf(msg, args...))
	w.Write([]byte(msg))
	log.Println(msg)
}
