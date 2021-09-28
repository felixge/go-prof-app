package main

import (
	"reflect"
	"runtime/metrics"
	"testing"
)

func Test_histDist_Update(t *testing.T) {
	tests := []struct {
		In   []*metrics.Float64Histogram
		Want [][]histEvent
	}{
		{
			In: []*metrics.Float64Histogram{
				{
					Counts:  []uint64{2, 7, 10, 3, 1},
					Buckets: []float64{1, 11, 21, 31, 41, 51},
				},
				{
					Counts:  []uint64{3, 7, 10, 5, 1},
					Buckets: []float64{1, 11, 21, 31, 41, 51},
				},
			},
			Want: [][]histEvent{
				{
					{float64(1+11) / 2, 2},
					{float64(11+21) / 2, 7},
					{float64(21+31) / 2, 10},
					{float64(31+41) / 2, 3},
					{float64(41+51) / 2, 1},
				},
				{
					{float64(1+11) / 2, 1},
					{float64(31+41) / 2, 2},
				},
			},
		},
	}

	for _, test := range tests {
		hd := &histDist{}
		for i, hg := range test.In {
			got := hd.Update(hg)
			if !reflect.DeepEqual(got, test.Want[i]) {
				t.Fatalf("%d: got=%v want=%v", i, got, test.Want)
			}
		}
	}

}
