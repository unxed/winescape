//go:build !((!windows && !linux && !freebsd) || (!amd64 && !arm64))

package winescape

// haveSyscallTable reports whether a real syscall number table is compiled in.
// The build constraint is the negation of numbers_fallback.go's.
const haveSyscallTable = true
