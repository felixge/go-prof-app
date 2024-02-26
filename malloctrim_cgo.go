//go:build linux && cgo
// +build linux,cgo

package main

import "time"

/*
#include <malloc.h>

#ifdef __GLIBC__
  int isGlibc = 1;
#else
  int isGlibc = 0;
  int malloc_trim(size_t __pad) {
    return 0;
  }
#endif
*/
import "C"

// mallocTrimEvery executes the malloc_trim function on the given interval. It
// does nothing if the platform is not linux, or the platform is not based on
// GlibC (since malloc_trim is a GlibC extension). The purpose of this call is
// to return unused pages to the system, instead of keeping them in the app's
// RSS.
func mallocTrimEvery(freq time.Duration) {
	if C.isGlibc == 0 {
		// Not GLIBC, so no malloc_trim...
		return
	}
	go func() {
		for {
			time.Sleep(freq)
			C.malloc_trim(0)
		}
	}()
}
