//go:build !linux || !cgo
// +build !linux !cgo

package main

import "time"

// mallocTrimEvery executes the malloc_trim function on the given interval. It
// does nothing if the platform is not linux, or the platform is not based on
// GlibC (since malloc_trim is a GlibC extension). The purpose of this call is
// to return unused pages to the system, instead of keeping them in the app's
// RSS.
func mallocTrimEvery(time.Duration) {
	// No malloc_trim is available on this platform...
}
