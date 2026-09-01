# libwinescape

Raw POSIX syscall trampolines for Windows binaries running under Wine.

## Overview

`libwinescape` is a lightweight, zero-dependency library (in pure Go assembly and C) that allows Windows-compiled binaries (`GOOS=windows` or MinGW/MSVC) running under Wine to bypass Win32/NTDLL emulation overhead and issue POSIX system calls directly to the host operating system kernel.

**Scope:** this library exists to make Wine-hosted programs feel and perform
like their native POSIX counterparts, for file-manager-shaped tools
specifically — not as a general POSIX-for-Windows layer. See
[`docs/SCOPE.md`](docs/SCOPE.md) for the enforceable boundary, including
which parts of the API below are borderline and why.

Unlike approaches that try to dynamically link host ELF `.so` libraries (which is fundamentally impossible from pure PE binaries under modern Wine without cgo or custom hypervisors), `libwinescape` invokes the kernel's raw trap instruction (`SYSCALL` on x86-64, `SVC #0` on ARM64) with native host register calling conventions.

## Platform Support Matrix

| Host OS under Wine | Direct Raw Syscalls Supported | Status |
|---|---|---|
| **Linux (amd64, arm64)** | Yes (native kernel ABI) | Fully supported |
| **FreeBSD (amd64)** | Yes (carry-flag error return ABI) | Supported |
| **NetBSD, DragonFly** | Yes (presumed BSD ABI) | Pending live hardware verification |
| **OpenBSD** | **No** (`pledge`/`msyscall` requires syscalls to originate within `libc.so`) | Out of scope |
| **macOS (Darwin)** | **No** (unstable syscall numbers; `libSystem` required) | Out of scope |
| **Illumos / Solaris** | **No** (unstable kernel syscall ABI; `libc` required) | Out of scope |

> **Warning:** This library is specifically designed for code compiled for Windows that executes in a Wine environment. On native Windows, calls will fail safely without crashing.

**`Available()` runs a real self-test, not just an OS-name check.** The first
call round-trips actual data through open/write/read/close/unlink on a real
file, plus a deliberate missing-file check that confirms errno translation
itself works, before reporting `true` -- so a future Wine version that broke
the raw-syscall path would make `Available()` report `false` automatically,
rather than silently corrupting data until a human noticed. `SelfTestError()`
returns why, if it's ever relevant to log. See
[`docs/RISK-REVIEW-farmanager-1123.md`](docs/RISK-REVIEW-farmanager-1123.md)
item 4 for what this does and doesn't cover.

**Threading and signal safety under Wine:** see
[`docs/threading.md`](docs/threading.md) and
[`docs/STATUS-signal-safety-review.md`](docs/STATUS-signal-safety-review.md)
for what's been verified (by reading Wine/Go runtime source and by soak
testing under real Wine) versus what remains a standing, monitored risk —
including a response to external critique of this approach in
[`docs/RISK-REVIEW-farmanager-1123.md`](docs/RISK-REVIEW-farmanager-1123.md).

## Architecture

1. **Generic Trampoline Layer:** A minimal assembly dispatcher per (OS, architecture, language) that maps up to 6 arguments into host registers and captures error status.
2. **Syscall Table (Single Source of Truth):** Defined in `spec/table.go`, verified against upstream OS kernel headers. Code generator in `cmd/gen-numbers` generates Go constants and C `#define` headers.
3. **POSIX Wrappers:** Typed wrappers for standard filesystem operations (`open`, `read`, `write`, `close`, `fstat`, `lseek`, `getdents64`, `unlink`, `rename`, `mkdir`, `readlink`, `access`, etc.).
4. **Optional Runtime Scheduler Integration (`go/gort`):** Dedicated `LockOSThread` worker pool to prevent Go runtime scheduler starvation during blocking syscalls.
## API Catalog

- **POSIX File I/O & Metadata:** `Open`, `Openat`, `Read`, `Write`, `Pread`, `Pwrite`, `Seek`, `Close`, `Fstat`, `Fstatat`, `Stat`, `Lstat`, `Unlink`, `Unlinkat`, `Rmdir`, `Mkdir`, `Mkdirat`, `Rename`, `Renameat`, `Renameat2`/`RenameNoReplace` (atomic no-clobber rename, Linux only), `Symlink`, `Symlinkat`, `Readlink`, `Readlinkat`, `Access`, `Faccessat`, `Chmod`, `Fchmodat`, `Chown`, `Fchownat`, `Lchown`, `Chtimes`, `Utimensat`, `Truncate`, `Ftruncate`.
- **High-Level VFS Operations:** `ReadFile`, `WriteFile`, `MkdirAll`, `RemoveAll`, `CopyFile` (with in-kernel zero-copy `copy_file_range`), `CreateTemp`, `ToUnixPath` (instant zero-syscall path normalization).
- **Fast Directory Scanning:** `Getdents64`, `ParseDirent64`, `ReadDir` (chunked 64KB kernel buffer reader).
- **Standard `io/fs.FS` Integration:** `DirFS(root)` providing `fs.FS`, `fs.StatFS`, `fs.ReadDirFS`, `fs.ReadFileFS`, `FileInfo`, `DirEntry`.
- **Pipes & Memory Mapping:** `Pipe2`, `Dup3`, `Mmap`, `Munmap`.
- **Real-Time Linux File Watching:** `InotifyInit1`, `InotifyAddWatch`, `InotifyRmWatch`, `ParseInotifyEvents`.
- **POSIX Host Identity:** `Getuid`, `Getgid`, `Geteuid`, `Getegid`, `Getppid`.
- **Process Signals & Control:** `Kill`, `Wait4`, `Execve`.
- **Terminal Control & Clocks:** `Ioctl`, `GetWinsize` (`TIOCGWINSZ`), `Tcgetattr`, `Tcsetattr`, `MakeRaw`, `ClockGettime`, `ClockNanosleep`, `Sleep` (with automatic signal restart on `EINTR`).
- **Raw POSIX Networking & IPC:** `Socket`, `Bind`, `Connect`, `Listen`, `Accept4`, `DialUnix`, `ListenUnix` (connect directly to host X11, Wayland, D-Bus, Docker, ssh-agent sockets).
- **Go Scheduler Isolation (`gort`):** Dedicated `LockOSThread` worker pool (`gort.NewPool()`, `gort.RunInPool()`).

## VFS Integration Example

```go
package vfs

import (
	winescape "github.com/unxed/libwinescape/go"
)

func ReadDirectory(path string) ([]FileItem, error) {
	if winescape.Available() {
		// Fast path: bypass wineserver, read directory in bulk via raw getdents64
		entries, err := winescape.ReadDir(path)
		if err == nil {
			return convertEntries(entries), nil
		}
	}
	// Fallback path: standard Windows / Win32 traversal
	return fallbackReadDirectory(path)
}
## Testing & Verification

- **Portable unit tests (any host OS without Wine):**
  ```bash
  make test
  ```
- **Live POSIX syscall integration tests under Wine:**
  ```bash
  make test-wine
  ```
- **Standalone raw syscall probe under Wine:**
  ```bash
  make probe
  wine probe/wine_syscall_probe.exe
  ```

## Similar projects

* [WinUnixPtyBridge](https://gitlab.howett.net/DHowett/WinUnixPtyBridge)

