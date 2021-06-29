package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/jackc/pgx/v4/stdlib"
	sqltrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/database/sql"
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/julienschmidt/httprouter"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"
)

var (
	//go:embed version.txt
	rawVersion string
	version    = strings.TrimSpace(rawVersion)
	//go:embed schema.sql
	schemaSQL string
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func envWithDefault(key string, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
		maxConnsF      = flag.Int("maxConns", 20, "Max number of database connections.")
		serviceF       = flag.String("dd.service", envWithDefault("DD_SERVICE", "go-prof-app"), "Name of the service.")
		envF           = flag.String("dd.env", envWithDefault("DD_ENV", "dev"), "Name of the environment the app is running in")
		powDifficultyF = flag.Int("powDifficulty", 4, "Difficulty level for pow")
		ddKey          = flag.String("dd.key", "", "API key for dd-trace-go agentless profile uploading")
		ddPeriod       = flag.Duration("dd.period", profiler.DefaultPeriod, "Profiling period for dd-trace-go")
		ddCPUDuration  = flag.Duration("dd.cpuDuration", profiler.DefaultDuration, "CPU duration for dd-trace-go")
		ddProfiler     = flag.Bool("dd.profiler", true, "Enable dd-trace-go profiler")
		ddTracer       = flag.Bool("dd.tracer", true, "Enable dd-trace-go tracer")
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

	log.Printf("Starting up %s version %s at http %s", *serviceF, version, *addrF)
	var profilesS []string
	for _, p := range profiles {
		profilesS = append(profilesS, p.String())
	}

	if !*ddProfiler {
		log.Printf("Not starting profiler because its disabled")
	} else {
		log.Printf("Starting profiler with: %v", profilesS)

		// addr comes from DD_AGENT_HOST
		statsd, err := statsd.New("")
		if err != nil {
			return err
		}

		profilerOptions := []profiler.Option{
			profiler.WithService(*serviceF),
			profiler.WithEnv(*envF),
			profiler.WithVersion(version),
			profiler.WithProfileTypes(profiles...),
			profiler.CPUDuration(*ddCPUDuration),
			profiler.WithPeriod(*ddPeriod),
			profiler.WithStatsd(statsd),
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
	}

	if !*ddTracer {
		log.Printf("Profiling disabled, not starting profiler")
	} else {
		log.Printf("Starting tracer")
		tracer.Start(
			tracer.WithEnv(*envF),
			tracer.WithService(*serviceF),
			tracer.WithServiceVersion(version),
		)
		defer tracer.Stop()
	}

	sqltrace.Register("pgx", stdlib.GetDefaultDriver())
	db, err := sqltrace.Open("pgx", "postgres://")
	if err != nil {
		return err
	} else if _, err := db.Exec(schemaSQL); err != nil {
		log.Printf("Failed to to apply schema.sql: %s", err)
	} else {
		log.Printf("Applied schema.sql")
	}

	db.SetMaxOpenConns(*maxConnsF)

	router := httptrace.New()
	router.Handler("GET", "/", VersionHandler{Version: version})
	// Accept GET/POST for transaction endpoint so one can hit it more easily
	router.Handler("GET", "/transaction", TransactionHandler{DB: db, PowDifficultiy: *powDifficultyF})
	router.Handler("POST", "/transaction", TransactionHandler{DB: db, PowDifficultiy: *powDifficultyF})
	return http.ListenAndServe(*addrF, router)
}
