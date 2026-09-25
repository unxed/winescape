//go:build (!windows && !linux && !freebsd) || (!amd64 && !arm64)

package winescape

// numbers_fallback.go is in effect: every syscall number is 0 because this
// GOOS/GOARCH (e.g. darwin, openbsd) is not a supported host (see README).
const haveSyscallTable = false
