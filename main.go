package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"runtime/metrics"
	"runtime/trace"
	"strings"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/jackc/pgx/v4/stdlib"
	sqltrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/database/sql"
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/julienschmidt/httprouter"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"
)

var (
	version = "n/a"
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
		traceF         = flag.String("trace", "", "Capture execution trace to file.")
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

	if *traceF != "" {
		log.Printf("Capturing executiong trace to %q", *traceF)
		traceFile, err := os.Create(*traceF)
		if err != nil {
			return err
		}
		defer traceFile.Close()

		if err := trace.Start(traceFile); err != nil {
			return err
		}
		defer trace.Stop()
	}

	var profilesS []string
	for _, p := range profiles {
		profilesS = append(profilesS, p.String())
	}

	// addr comes from DD_AGENT_HOST
	statsd, err := statsd.New("")
	if err != nil {
		log.Printf("Failed to init statsd client: %s", err)
	} else {
		go reportMemstats(statsd)
		go reportRuntimeMetrics(statsd)
	}

	if !*ddProfiler {
		log.Printf("Not starting profiler because its disabled")
	} else {
		log.Printf("Starting profiler with: %v", profilesS)

		profilerOptions := []profiler.Option{
			profiler.WithService(*serviceF),
			profiler.WithEnv(*envF),
			profiler.WithVersion(version),
			profiler.WithProfileTypes(profiles...),
			profiler.CPUDuration(*ddCPUDuration),
			profiler.WithPeriod(*ddPeriod),
			profiler.WithStatsd(statsd),
			profiler.WithTags("go_version:" + runtime.Version()),
		}
		if *ddKey != "" {
			log.Printf("Using agentless uploading")
			profilerOptions = append(
				profilerOptions,
				profiler.WithAPIKey(*ddKey),
				profiler.WithAgentlessUpload(),
			)
		}
		if err := profiler.Start(profilerOptions...); err != nil {
			return err
		}
		defer profiler.Stop()
	}

	if !*ddTracer {
		log.Printf("Tracing disabled, not starting tracer")
	} else {
		log.Printf("Starting tracer")
		tracer.Start(
			tracer.WithEnv(*envF),
			tracer.WithService(*serviceF),
			tracer.WithServiceVersion(version),
			tracer.WithGlobalTag("go_version", runtime.Version()),
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
	router.Handler("GET", "/io-bound", &PostsHandler{
		DB:          db,
		CPUDuration: 10 * time.Millisecond,
		SQLDuration: 90 * time.Millisecond,
	})
	router.Handler("GET", "/cpu-bound", &PostsHandler{
		DB:          db,
		CPUDuration: 90 * time.Millisecond,
		SQLDuration: 10 * time.Millisecond,
	})
	router.Handler("GET", "/cgo-cpu-bound", &PostsHandler{
		DB:          db,
		CPUDuration: 90 * time.Millisecond,
		SQLDuration: 10 * time.Millisecond,
		CGO:         true,
	})
	// Accept GET/POST for transaction endpoint so one can hit it more easily
	router.Handler("GET", "/transaction", TransactionHandler{DB: db, PowDifficultiy: *powDifficultyF})
	router.Handler("POST", "/transaction", TransactionHandler{DB: db, PowDifficultiy: *powDifficultyF})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go http.ListenAndServe(*addrF, router)

	sig := <-sigCh
	log.Printf("Received %s, shutting down", sig)
	return nil
}

func reportMemstats(statsd *statsd.Client) {
	for {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)
		statsd.Gauge("go.memstats.alloc", float64(stats.Alloc), nil, 1)
		statsd.Gauge("go.memstats.buckhashsys", float64(stats.BuckHashSys), nil, 1)
		statsd.Gauge("go.memstats.frees", float64(stats.Frees), nil, 1)
		statsd.Gauge("go.memstats.gccpufraction", float64(stats.GCCPUFraction), nil, 1)
		statsd.Gauge("go.memstats.gcsys", float64(stats.GCSys), nil, 1)
		statsd.Gauge("go.memstats.heapalloc", float64(stats.HeapAlloc), nil, 1)
		statsd.Gauge("go.memstats.heapidle", float64(stats.HeapIdle), nil, 1)
		statsd.Gauge("go.memstats.heapinuse", float64(stats.HeapInuse), nil, 1)
		statsd.Gauge("go.memstats.heapobjects", float64(stats.HeapObjects), nil, 1)
		statsd.Gauge("go.memstats.heapreleased", float64(stats.HeapReleased), nil, 1)
		statsd.Gauge("go.memstats.heapsys", float64(stats.HeapSys), nil, 1)
		statsd.Gauge("go.memstats.lastgc", float64(stats.LastGC), nil, 1)
		statsd.Gauge("go.memstats.lookups", float64(stats.Lookups), nil, 1)
		statsd.Gauge("go.memstats.mcacheinuse", float64(stats.MCacheInuse), nil, 1)
		statsd.Gauge("go.memstats.mcachesys", float64(stats.MCacheSys), nil, 1)
		statsd.Gauge("go.memstats.mspaninuse", float64(stats.MSpanInuse), nil, 1)
		statsd.Gauge("go.memstats.mspansys", float64(stats.MSpanSys), nil, 1)
		statsd.Gauge("go.memstats.mallocs", float64(stats.Mallocs), nil, 1)
		statsd.Gauge("go.memstats.numforcedgc", float64(stats.NumForcedGC), nil, 1)
		statsd.Gauge("go.memstats.numgc", float64(stats.NumGC), nil, 1)
		statsd.Gauge("go.memstats.othersys", float64(stats.OtherSys), nil, 1)
		statsd.Gauge("go.memstats.pausetotalns", float64(stats.PauseTotalNs), nil, 1)
		statsd.Gauge("go.memstats.stackinuse", float64(stats.StackInuse), nil, 1)
		statsd.Gauge("go.memstats.stacksys", float64(stats.StackSys), nil, 1)
		statsd.Gauge("go.memstats.sys", float64(stats.Sys), nil, 1)
		statsd.Gauge("go.memstats.totalalloc", float64(stats.TotalAlloc), nil, 1)
		time.Sleep(10 * time.Second)
	}
}

func reportMetrics(statsd statsd.ClientInterface) {
	descs := metrics.All()
	samples := make([]metrics.Sample, len(descs))
	for i := range samples {
		samples[i].Name = descs[i].Name
	}

	for {
		metrics.Read(samples)
		for _, sample := range samples {
			name, value := sample.Name, sample.Value
			name, unit := metricName(name)

			switch value.Kind() {
			case metrics.KindUint64:
				statsd.Gauge(name+"."+unit, float64(value.Uint64()), nil, 1)
			case metrics.KindFloat64:
				statsd.Gauge(name+"."+unit, value.Float64(), nil, 1)
			case metrics.KindFloat64Histogram:
				//if name == "runtime.go.metrics.sched.latencies" {
				//for _, v := range value.Float64Histogram().Buckets {
				//if math.IsInf(v, 0) {
				//continue
				//}
				//fmt.Printf("%s ", time.Duration(v*float64(time.Second)))
				//}
				//fmt.Printf("\n---------------------\n")
				//}
				stats := newHistStats(value.Float64Histogram())
				statsd.Gauge(name+".avg."+unit, stats.Avg, nil, 1)
				statsd.Gauge(name+".min."+unit, stats.Min, nil, 1)
				statsd.Gauge(name+".max."+unit, stats.Max, nil, 1)
				statsd.Gauge(name+".median."+unit, stats.Median, nil, 1)
				statsd.Gauge(name+".p95."+unit, stats.P95, nil, 1)
				statsd.Gauge(name+".p99."+unit, stats.P99, nil, 1)
			case metrics.KindBad:
				// This should never happen because all metrics are supported
				// by construction.
				panic("bug in runtime/metrics package!")
			default:
				// This may happen as new metrics get added.
				//
				// The safest thing to do here is to simply log it somewhere
				// as something to look into, but ignore it for now.
				// In the worst case, you might temporarily miss out on a new metric.
				fmt.Printf("%s: unexpected metric Kind: %v\n", name, value.Kind())
			}
		}
		time.Sleep(10 * time.Second)
	}
}

func newHistStats(h *metrics.Float64Histogram) (s histStats) {
	// TODO(fg) very inefficent implementation
	s.Avg = histAvg(h)
	s.Min = histPercentile(h, 0)
	s.Median = histPercentile(h, 0.5)
	s.P95 = histPercentile(h, 0.95)
	s.P99 = histPercentile(h, 0.99)
	s.Max = histPercentile(h, 1)
	return
}

type histStats struct {
	Avg    float64
	Min    float64 // aka P0
	Median float64 // aka P50
	P95    float64
	P99    float64
	Max    float64 // aka P100
}

// see https://www.statology.org/histogram-mean-median/
func histAvg(h *metrics.Float64Histogram) float64 {
	var sum float64
	var count float64
	for i, val := range h.Counts {
		min, max := h.Buckets[i], h.Buckets[i+1]
		// TODO(fg) is there a better way?
		if math.IsInf(min, 0) || math.IsInf(max, 0) {
			continue
		}
		sum += float64(val) * (min + max) / 2
		count += float64(val)
	}
	return sum / count
}

// see https://stats.stackexchange.com/a/65718
func histPercentile(h *metrics.Float64Histogram, p float64) float64 {
	var countSum uint64
	for i, count := range h.Counts {
		min, max := h.Buckets[i], h.Buckets[i+1]
		// TODO(fg) is there a better way?
		if math.IsInf(min, 0) || math.IsInf(max, 0) {
			continue
		}
		countSum += count
	}
	var countCum uint64
	var min, max float64
	for i, count := range h.Counts {
		min, max = h.Buckets[i], h.Buckets[i+1]
		// TODO(fg) is there a better way?
		if math.IsInf(min, 0) || math.IsInf(max, 0) {
			continue
		}

		if p == 0 {
			return min
		}

		countCum += count
		if float64(countCum) >= float64(countSum)*p {
			break
		}
	}
	if p == 1 {
		return max
	}
	return (min + max) / 2
}

var metricRegex = regexp.MustCompile("^(?P<name>/[^:]+):(?P<unit>[^:*/]+(?:[*/][^:*/]+)*)$")

func metricName(runtimeName string) (string, string) {
	m := metricRegex.FindStringSubmatch(runtimeName)
	if len(m) != 3 {
		return "", ""
	}
	return "runtime.go.metrics." + strings.ReplaceAll(m[1][1:], "/", "."), m[2]
}
