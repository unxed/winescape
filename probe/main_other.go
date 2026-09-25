//go:build !amd64

package main

import (
	"fmt"
	"os"
	"runtime"
)

// The probe issues a raw x86-64 SYSCALL (probe_amd64.s); there is no
// implementation for other architectures.
func main() {
	fmt.Fprintf(os.Stderr, "probe: only implemented for amd64 (this is %s/%s)\n", runtime.GOOS, runtime.GOARCH)
	os.Exit(2)
}
