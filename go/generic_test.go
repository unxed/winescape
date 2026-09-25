package winescape

import (
	"runtime"
	"testing"
)

// skipWithoutSyscallTable skips tests that check syscall numbers on host
// platforms libwinescape does not support (numbers_fallback.go, all zero).
func skipWithoutSyscallTable(t *testing.T) {
	t.Helper()
	if !haveSyscallTable {
		t.Skipf("no syscall number table for %s/%s: unsupported host platform (see README Platform Support Matrix)", runtime.GOOS, runtime.GOARCH)
	}
}

func TestGenericSyscall_NumbersDefined(t *testing.T) {
	skipWithoutSyscallTable(t)
	// Verify that key syscall numbers are present in the generic dispatch table.
	if sysGetpid == 0 {
		t.Errorf("sysGetpid must be non-zero on supported target platforms")
	}
	if sysRead == 0 && sysWrite == 0 && sysClose == 0 {
		t.Errorf("expected basic I/O syscall numbers to be initialized")
	}
}
