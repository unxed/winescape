# Adding a New Syscall to libwinescape

To add a new syscall:

1. **Verify Numbers from Primary Sources:**
   Look up the exact syscall number in official kernel headers:
   - Linux x86-64: `asm/unistd_64.h`
   - Linux ARM64: `asm-generic/unistd.h`
   - FreeBSD amd64: `sys/syscall.h`
   *Never guess or infer numbers.*

2. **Add to `spec/table.go`:**
   Add an entry to `SyscallTable` with the argument count and verified numbers per platform.

3. **Regenerate Tables:**
   Run:
   ```bash
   make gen   # go run ./cmd/gen-numbers . && gofmt -w go/numbers_*.go
   ```
   The generator does not align the const blocks; the committed files are gofmt-ed,
   and CI regenerates them and fails on any difference.
   This updates:
   - `go/numbers_linux_amd64.go`
   - `go/numbers_linux_arm64.go`
   - `go/numbers_freebsd_amd64.go`
   - `c/include/winescape_numbers.h`

4. **Implement High-Level Wrappers:**
   Add typed wrapper functions in `go/fs.go` and `c/src/fs.c` handling argument marshaling and NUL-terminated strings.

5. **Write Tests:**
   Add portable unit tests for argument marshaling and errno translation in `go/fs_test.go`.
