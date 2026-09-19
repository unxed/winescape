# Scope

`libwinescape` exists to serve one concrete goal: make a Windows-compiled
binary running under Wine **feel and perform like the native POSIX build of
the same program**, for the class of program that cares about this — file
managers and similar tools (Far Manager 3, `f4`) first, anything else built
on the same premise second. It is not a general-purpose POSIX-for-Windows
compatibility layer, and it is not trying to become one. This document is the
enforceable boundary that keeps it from drifting into that instead.

## The test

A change belongs in this library if and only if it satisfies at least one of:

1. **It measurably changes what the user sees or feels under Wine**, in a
   file/terminal-manager context specifically: navigation speed on large
   directories, whether a path shows as `/home/user/...` or
   `Z:\home\user\...`, whether file permissions/ownership can be shown or
   edited at all (Win32 has no representation for `rwxrwxrwx`/SUID/UID/GID —
   this isn't slower through Win32, it's *impossible* through Win32), or
   whether a terminal panel behaves like a real POSIX shell instead of
   `cmd.exe` wrapped in `wineconsole`.
2. **It works around a genuine Wine bug or limitation**, where going straight
   to the kernel sidesteps a problem that exists specifically because of the
   Win32→Wine→POSIX translation layer (console rendering glitches, the
   directory-enumeration IPC round-trip through `wineserver`, etc.) — not a
   POSIX feature Windows also happens to lack in general.
3. **It provides host capability Win32 cannot reach at all**, not merely
   slower — connecting to a host `AF_UNIX` socket (X11 display, Wayland
   compositor, D-Bus session bus) has no Winsock equivalent whatsoever; this
   is a capability gap, not a performance one, and it's the concrete mechanism
   behind `f4`'s own planned native-GUI direction (WINE.md Part C).

A change does **not** belong here on the strength of "POSIX has this and we
don't" alone. Completeness is not a goal. If a capability doesn't trace back
to one of the three tests above for an actual file-manager/terminal-manager
use case, it doesn't belong in this library even if it's a natural POSIX
syscall to wrap.

## Applying the requester's own criteria directly

- **Speed, path/slash correctness, Wine-bug workarounds, file
  permissions** — explicitly in scope; these are tests 1 and 2 above,
  verbatim.
- **Networking that already works the same way through plain WinAPI/Winsock**
  — out of scope, *unless* it measurably affects speed (test 1), in which
  case it's in scope for that reason, not because it's networking.
  Raw `AF_UNIX` to host sockets Winsock cannot reach at all (test 3) is a
  different case entirely and stays in scope regardless of speed.
- **POSIX surface pulled in for its own sake, disconnected from a concrete
  UX improvement** — explicitly out of scope. This is the failure mode this
  document exists to prevent.

## Classifying what's already in the repository

The API surface grew past what a scope document existed to constrain — this
table is the first pass at applying the test above honestly, including where
it says an existing feature doesn't clearly pass.

| Area | Verdict | Why |
|---|---|---|
| `Open/Read/Write/Pread/Pwrite/Close/Seek`, `Stat/Lstat/Fstat`, `ReadDir`/`Getdents64` | **In scope** | Test 1 (speed) and test 2 (bypasses `wineserver` round-trips) — this is the core of what `f4`'s `vfs/hostfs` actually calls. |
| `Chmod/Chown/Lchown/Chtimes`, `Symlink/Readlink`, `Mkdir/Rmdir/Rename/Unlink` | **In scope** | Test 1 — Win32 cannot represent POSIX permissions/ownership/symlinks at all; this is capability, not speed. |
| `Renameat2`/`RenameNoReplace` (`RENAME_NOREPLACE`) | **In scope** | Test 1 — atomic no-clobber rename has no race-free userspace equivalent (a check-then-rename always has a TOCTOU gap); `f4`'s own real-POSIX build already relies on exactly this (`vfs/rename_noreplace_linux.go`), so posix mode needs it to behave the same way, not merely similarly. |
| `copy_file_range`/`CopyFile` (kernel zero-copy) | **In scope** | Test 1 — directly affects perceived copy speed on same-filesystem operations, the kind of thing a file manager's users notice. |
| `InotifyInit1`/`InotifyAddWatch`/`ParseInotifyEvents` | **In scope** | Test 1/2 — replaces polling or `ReadDirectoryChangesW` translated through Wine; real-time panel refresh is a genuine, visible UX property. |
| `Ioctl`/`TIOCGWINSZ`/`Tcgetattr`/`Tcsetattr`/`MakeRaw`, `Execve`/`Pipe2` for running a native shell | **In scope** | Test 1/2 — this is precisely what lets an embedded terminal be a real POSIX shell instead of `cmd.exe` under `wineconsole`, which WINE.md documents as an actual source of bugs in `f4`. |
| `Getuid/Getgid/Geteuid/Getegid/Getppid` | **In scope** | Test 1 — needed to display/compare real UID/GID for permission rendering (test 1's ownership case), not identity for its own sake. |
| `DialUnix`/`ListenUnix`, raw `AF_UNIX` (X11/Wayland/D-Bus/ssh-agent/docker.sock) | **In scope, narrowly** | Test 3 — genuine capability gap, and directly the mechanism WINE.md Part C already commits to for native GUI. Scope is the *unix-domain* socket case specifically, not POSIX networking generally — see next row. |
| `Socket/Bind/Connect/Listen/Accept4` for general (non-`AF_UNIX`) networking | **Borderline — plausible, not yet measured** | Ordinary Winsock-via-Wine is not obviously slow for typical file-manager network use (small control-plane traffic, moderate-throughput transfers), and test 1 hasn't been demonstrated for that case. A concrete scenario where it plausibly *would* matter: a very high-throughput link (the cited example: 100Gbit/s) doing sustained bulk transfer through `f4`'s network VFS providers (SFTP/FTP/FISH), where even small per-call overhead in Wine's Winsock emulation could become the bottleneck at that throughput. Plausible, not confirmed -- Wine's actual `ws2_32` data-path overhead for already-connected sockets hasn't been measured here against raw `AF_INET`/`AF_INET6`. **Graduation bar, so this isn't added speculatively: benchmark real sustained throughput/latency, Win32 Winsock-through-Wine vs. raw sockets, under a realistic bulk-transfer load, and cite the number here.** Until then this stays borderline, kept only because it shares code with the already-in-scope `DialUnix` case. |
| `Mmap`/`Munmap` | **In scope, narrowly — random-access I/O on large files** | Test 1 (speed/capability). Worth being precise about the justification: the value isn't "bridging Win32 and POSIX worlds" -- there is only one address space (the PE process's own), so a `[]byte` filled via raw `Read`/`Pread` is already directly usable by anything else in the same process without any bridge. The real justification is narrower and still real: `mmap` avoids the kernel-to-userspace copy `read`/`pread` always pay, and gives random access without repeated `lseek`+`read` pairs -- both of which matter specifically for a large-file viewer/editor doing random-access reads on multi-GB files, not for a linear copy. Kept in scope on that specific basis; any other use should be named here too, not assumed to be covered by this one. |
| `Kill`/`Wait4` | **In scope, narrowly** | Needed to manage a child shell/process launched via `Execve` (test 2's terminal case) — process-group lifecycle for something this library itself spawns, not general process control. |
| `Spawn`, `StartPTY`/`OpenPTY`, `Poll`, `SetWinsize`, `Tcgetpgrp` (Linux hosts) | **In scope** | Tests 1 and 2 -- the terminal case named above, finished. `Execve` replaces the calling process, so without a fork there was no way to run a real POSIX shell in an embedded terminal at all; the earlier roadmap assessment deferred a `forkpty`-shaped helper until the first real caller existed, and that caller is `f4`'s built-in terminal under Wine. Deliberately narrow: `Spawn` takes the few things a terminal needs (descriptors, `setsid`, controlling terminal, working directory, environment), not the whole `posix_spawn` attribute set. |
| Generic dispatch (`WS_CALL0..6` / `winescape.SyscallN`/`Call`) | **In scope as an escape hatch, with a usage rule** | Exists so a caller isn't blocked waiting for a named wrapper for a syscall this library hasn't gotten to yet. **Rule: this is for prototyping and for genuinely one-off needs, not a substitute for adding a real entry to `spec/table.go` once a capability is used more than once.** A PR that adds a new *named, permanent* API surface via raw dispatch instead of the table-driven path should be treated as not having gone through scope review at all. |

## What would get a "no"

Concrete, so this isn't abstract: a PR adding `Rlimit`/`Setrlimit`,
`ptrace`, full `sched_setaffinity` CPU-affinity control, or a generic
`Fcntl` surface covering every possible flag, with the justification "POSIX
has it and we might need it later," would fail every test above and should be
declined. If a future concrete need shows up (a real file-manager feature
that requires one of these), the request should name that feature and the
specific narrow syscall surface it needs — not the syscall's full generality.

## Process

- Every addition to the public API (a new wrapper in `go/`/`c/src/`, not a
  generic-dispatch one-off) should be checked against the three tests above
  in its commit message or PR description — which test it passes and for
  which concrete `f4`/Far3 feature.
- This file is expected to be updated, not treated as final. The "borderline"
  rows above are the actual live edge of scope right now; resolving them
  (citing a use case, or removing the feature) is part of normal maintenance,
  not a one-time cleanup.
