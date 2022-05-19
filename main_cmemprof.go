//go:build experimental_cmemprof
// +build experimental_cmemprof

package main

// Only enable cgotraceback (which adds a 3rd-party dependency on libunwind) if
// we're profiling C allocations

import (
	_ "github.com/nsrip-dd/cgotraceback"
)
