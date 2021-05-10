package main

import (
	_ "embed"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/julienschmidt/httprouter"
)

var (
	//go:embed version.txt
	rawVersion string
	version    = strings.TrimSpace(rawVersion)
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var (
		addrF    = flag.String("addr", "localhost:8080", "Listen addr for http server")
		versionF = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	if *versionF {
		fmt.Printf("%s\n", version)
		return nil
	}

	router := httprouter.New()
	router.Handler("GET", "/version", VersionHandler{Version: version})
	return http.ListenAndServe(*addrF, router)
}
