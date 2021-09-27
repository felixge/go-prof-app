package main

import (
	"math"
	"runtime/metrics"
	"testing"
)

//func Test_metricName(t *testing.T) {
//for _, d := range metrics.All() {
//fmt.Printf("%s -> %s\n", d.Name, metricName(d.Name))
//}
//}

func Test_newHistStats(t *testing.T) {
	tests := []struct {
		In   *metrics.Float64Histogram
		Want histStats
	}{
		{
			In: &metrics.Float64Histogram{
				Counts:  []uint64{2, 7, 10, 3, 1},
				Buckets: []float64{1, 11, 21, 31, 41, 51},
			},
			Want: histStats{
				Avg:    23.39,
				Min:    1,
				Median: (21 + 31) / 2,
				P95:    (31 + 41) / 2,
				P99:    (41 + 51) / 2,
				Max:    51,
			},
		},
		{
			In: &metrics.Float64Histogram{
				Counts:  []uint64{100, 2, 7, 10, 3, 1, 100},
				Buckets: []float64{math.Inf(-1), 1, 11, 21, 31, 41, 51, math.Inf(1)},
			},
			Want: histStats{
				Avg:    23.39,
				Min:    1,
				Median: (21 + 31) / 2,
				P95:    (31 + 41) / 2,
				P99:    (41 + 51) / 2,
				Max:    51,
			},
		},
	}
	for _, test := range tests {
		got := newHistStats(test.In)
		if math.Abs(got.Avg-test.Want.Avg) > 0.1 {
			t.Fatalf("got=%f want=%f", got.Avg, test.Want.Avg)
		}
		if math.Abs(got.Min-test.Want.Min) > 0.1 {
			t.Fatalf("got=%f want=%f", got.Min, test.Want.Min)
		}
		if math.Abs(got.Max-test.Want.Max) > 0.1 {
			t.Fatalf("got=%f want=%f", got.Max, test.Want.Max)
		}
		if math.Abs(got.Median-test.Want.Median) > 0.1 {
			t.Fatalf("got=%f want=%f", got.Median, test.Want.Median)
		}
		if math.Abs(got.P95-test.Want.P95) > 0.1 {
			t.Fatalf("got=%f want=%f", got.P95, test.Want.P95)
		}
		if math.Abs(got.P99-test.Want.P99) > 0.1 {
			t.Fatalf("got=%f want=%f", got.P99, test.Want.P99)
		}
	}
}
