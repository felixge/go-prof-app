package main

import (
	"fmt"
	"math"
	"runtime/metrics"
	"time"

	"github.com/DataDog/datadog-go/statsd"
)

func reportRuntimeMetrics(statsd statsd.ClientInterface) {
	descs := metrics.All()
	samples := make([]metrics.Sample, len(descs))
	for i := range samples {
		samples[i].Name = descs[i].Name
	}

	m := map[string]*histDist{}
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
				key := name + "." + unit
				hd, ok := m[key]
				if !ok {
					hd = &histDist{}
					m[key] = hd
				}
				for _, e := range hd.Update(value.Float64Histogram()) {
					statsd.Distribution(key, e.Value, nil, float64(e.Count))
				}
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

// histDist converts metrics.Float64Histogram values into
type histDist struct {
	prev *metrics.Float64Histogram
}

type histEvent struct {
	Value float64
	Count uint64
}

func (h *histDist) Update(hg *metrics.Float64Histogram) (events []histEvent) {
	for i, count := range hg.Counts {
		min, max := hg.Buckets[i], hg.Buckets[i+1]
		if math.IsInf(min, 0) || math.IsInf(max, 0) {
			continue
		}
		diff := count
		if h.prev != nil {
			diff = count - h.prev.Counts[i]
		}
		if diff == 0 {
			continue
		}

		events = append(events, histEvent{
			Count: diff,
			Value: (min + max) / 2,
		})
	}
	h.prev = cloneFloat64Histogram(hg)
	return
}

func cloneFloat64Histogram(hg *metrics.Float64Histogram) *metrics.Float64Histogram {
	clone := &metrics.Float64Histogram{
		Counts:  make([]uint64, len(hg.Counts)),
		Buckets: make([]float64, len(hg.Buckets)),
	}
	for i, count := range hg.Counts {
		clone.Counts[i] = count
		clone.Buckets[i] = hg.Buckets[i]
		clone.Buckets[i+1] = hg.Buckets[i+1]
	}
	return clone
}

//func (h *metrics.Float64Histogram) (s histStats) {
//// TODO(fg) very inefficent implementation
//s.Avg = histAvg(h)
//s.Min = histPercentile(h, 0)
//s.Median = histPercentile(h, 0.5)
//s.P95 = histPercentile(h, 0.95)
//s.P99 = histPercentile(h, 0.99)
//s.Max = histPercentile(h, 1)
//return
//}
