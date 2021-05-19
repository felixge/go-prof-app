package main

import (
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"
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
		profiles          = []profiler.ProfileType{profiler.CPUProfile, profiler.HeapProfile}
		availableProfiles = map[string]profiler.ProfileType{
			"cpu":       profiler.CPUProfile,
			"heap":      profiler.HeapProfile,
			"block":     profiler.BlockProfile,
			"mutex":     profiler.MutexProfile,
			"goroutine": profiler.GoroutineProfile,
		}

		addrF          = flag.String("addr", "localhost:8080", "Listen addr for http server")
		envF           = flag.String("env", "dev", "Name of the environment the app is running in")
		powDifficultyF = flag.Int("powDifficulty", 5, "Difficulty level for pow")
		ddKey          = flag.String("dd.key", "", "API key for dd-trace-go agentless profile uploading")
		ddPeriod       = flag.Duration("dd.period", profiler.DefaultPeriod, "Profiling period for dd-trace-go")
		ddCPUDuration  = flag.Duration("dd.cpuDuration", profiler.DefaultDuration, "CPU duration for dd-trace-go")
		versionF       = flag.Bool("version", false, "Print version and exit")
	)
	flag.Func("dd.profiles", `Comma separated list of dd-trace-go profiles to enable (default "cpu,heap")`, func(val string) error {
		profiles = nil
		val = strings.TrimSpace(val)
		if val == "" {
			return nil
		}
		for _, name := range strings.Split(val, ",") {
			profile, ok := availableProfiles[name]
			if !ok {
				return fmt.Errorf("unknown profile type: %q", name)
			}
			profiles = append(profiles, profile)
		}
		return nil
	})
	flag.Parse()

	if *versionF {
		fmt.Printf("%s\n", version)
		return nil
	}

	log.Printf("Starting up at http %s", *addrF)
	var profilesS []string
	for _, p := range profiles {
		profilesS = append(profilesS, p.String())
	}
	log.Printf("Enabled profiles: %v", profilesS)

	profilerOptions := []profiler.Option{
		profiler.WithService("go-prof-app"),
		profiler.WithEnv(*envF),
		profiler.WithVersion(version),
		profiler.WithProfileTypes(profiles...),
		profiler.CPUDuration(*ddCPUDuration),
		profiler.WithPeriod(*ddPeriod),
	}
	if *ddKey != "" {
		log.Printf("Using agentless uploading")
		profilerOptions = append(
			profilerOptions,
			profiler.WithAPIKey(*ddKey),
			profiler.WithAgentlessUpload(),
		)
	}
	err := profiler.Start(profilerOptions...)
	if err != nil {
		return err
	}
	defer profiler.Stop()

	db, err := sql.Open("pgx", "postgres://")
	if err != nil {
		return err
	}

	router := httprouter.New()
	router.Handler("GET", "/", VersionHandler{Version: version})
	router.Handler("POST", "/transaction", TransactionHandler{DB: db, PowDifficultiy: *powDifficultyF})
	return http.ListenAndServe(*addrF, router)
}
