package main

import (
	"testing"
)

func Test_calculatePoW(t *testing.T) {
	tests := []struct {
		Difficulty int
		Data       string
	}{
		{1, "foo"},
		{2, "bar"},
		{3, "foobar"},
	}
	for _, test := range tests {
		pow, nonce := calculatePoW(test.Difficulty, test.Data)
		if len(pow) != 40 || !verifyPoW(test.Data, pow, test.Difficulty, nonce) {
			t.Fatalf("%d:%s produced %s", test.Difficulty, test.Data, pow)
		}
		// sanity check
		for i := 0; i < test.Difficulty; i++ {
			if pow[i] != '0' {
				t.Fatalf("%d:%s produced %s", test.Difficulty, test.Data, pow)
			}
		}
	}
}

func Benchmark_calculatePoW(b *testing.B) {
	for i := 0; i < b.N; i++ {
		calculatePoW(3, "foo")
	}
}
