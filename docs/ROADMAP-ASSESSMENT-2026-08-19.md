# Assessment: unreviewed ROADMAP.md draft (second model, relevance unchecked) — 2026-08-19

A `docs/ROADMAP.md` draft arrived alongside the FarManager #1123 critique,
attributed to a different model that hadn't audited this repository's actual
state before writing it. Checked against the real API surface (`README.md`'s
catalog, `go/*.go`) before deciding what to do with it, rather than assuming
it described a gap.

## Already implemented (the draft is redundant here, not wrong)

Nearly everything in the draft's "Advanced POSIX VFS Subsystem" and "POSIX
IPC & Host Integration" sections already exists:

- Bulk `getdents64` directory scanning — `Getdents64`/`ReadDir`.
- `chmod`/`chown`/`utimensat` permission and ownership editing — `Chmod`,
  `Chown`/`Lchown`, `Chtimes`/`Utimensat`.
- Symlink handling — `Symlink`/`Symlinkat`, `Readlink`/`Readlinkat`.
- Kernel zero-copy copying — `copy_file_range` via `CopyFile`.
- Real-time change notification — `InotifyInit1`/`InotifyAddWatch`/
  `ParseInotifyEvents`.
- `AF_UNIX` sockets to host services — `DialUnix`/`ListenUnix` (see
  `docs/SCOPE.md` for why this specific case, and not general networking,
  is the one that stays in scope).
- Native process execution and PTY-adjacent terminal control — `Execve`,
  `Pipe2`, `Ioctl`/`TIOCGWINSZ`, `Tcgetattr`/`Tcsetattr`/`MakeRaw`.

## Actually missing (real gaps, not yet decided whether to add)

- `statx` — not implemented; `Stat`/`Lstat`/`Fstat` (classic `stat(2)`
  family) cover the fields `f4`/Far3 currently use. Whether the extra fields
  `statx` exposes (btime, more precise attribute masks) pass the scope test
  is not yet decided — no concrete UX feature has asked for them.
- `mknod` (device/FIFO node creation) — not implemented. Displaying existing
  special files (already possible via `Stat`'s mode bits) is different from
  creating them; no cited file-manager use case yet for the latter.
- A dedicated `forkpty`-shaped helper — not implemented as a single call, but
  its constituent pieces (`Execve`, `Pipe2`, `Ioctl`/`TIOCGWINSZ`,
  `Tcgetattr`/`MakeRaw`) already exist separately. Whether a combined
  convenience wrapper is worth adding is a small, low-risk decision, not a
  scope question — deferred to whoever wires the first real embedded-terminal
  call site, since the right shape will be clearer from actual usage than
  from writing it speculatively now.
  **Update:** that call site is `f4`'s built-in terminal under Wine, and the
  helper now exists as `Spawn`/`StartPTY`. It needed a fork, which the pieces
  listed here could not provide; see `docs/threading.md`.

## Disposition

The draft's own example commit message and D-Bus/Docker-socket examples go
further than `docs/SCOPE.md` recommends staying (general host-service
integration beyond what a file manager's core UX needs). Not adopted as
written; `docs/ROADMAP.md` is not being added to this repository. The parts
worth keeping are recorded above and are either already done or explicitly
deferred pending a real use case, per `docs/SCOPE.md`'s process section.
