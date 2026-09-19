package winescape

import (
	"runtime"
	"unsafe"
)

// Spawn starts a host process without going through Win32: fork, set the child
// up with raw syscalls, execve. It is what lets an embedded terminal run a real
// POSIX shell on a pseudo-terminal under Wine (docs/SCOPE.md, the terminal
// case). Execve alone cannot do that: it replaces the calling process, and
// there is no other way to make a second one.
//
// Host Linux only. Where the clone/setsid/... numbers are absent (FreeBSD hosts,
// unsupported architectures) or the raw-syscall path is unavailable (native
// Windows), Spawn returns ErrUnavailable and starts nothing.
//
// What the child does between fork and execve is the point of the design, and
// the reason this is one function rather than a recipe of separate calls. In
// the child only the forking thread exists; the Go runtime's other threads, the
// scheduler and every lock they hold are copies frozen mid-flight. So the child
// touches none of them: it makes raw syscalls only, from a nosplit function, on
// data that was fully prepared before the fork (no allocation, no stack growth,
// no write barriers -- the same rules syscall.forkAndExecInChild follows, for
// the same reason). See docs/threading.md.

// maxSpawnFiles bounds SpawnAttr.Files. The child's descriptor table is rebuilt
// from a fixed-size array so that the code between fork and exec never has to
// index a slice.
const maxSpawnFiles = 16

// Linux constants used by Spawn, identical on amd64 and arm64.
const (
	sigSetMask   = 2    // SIG_SETMASK
	fSetFd       = 2    // F_SETFD
	fDupFdCloexc = 1030 // F_DUPFD_CLOEXEC

	// TIOCSCTTY makes the terminal on a descriptor the controlling terminal
	// of the calling session leader.
	TIOCSCTTY = 0x540E

	sigKill = 9
	sigStop = 19
	// Signal numbers 1..64 are reset to their default disposition in the
	// child; SIGKILL and SIGSTOP cannot be changed.
	lastSignal = 64
)

// SpawnAttr describes the child Spawn starts. It mirrors the parts of
// syscall.SysProcAttr and ProcAttr a terminal needs.
type SpawnAttr struct {
	// Dir is the child's working directory. Empty leaves it unchanged.
	Dir string

	// Env is the child's environment as KEY=VALUE strings. Nil gives it an
	// empty environment; it is never inherited implicitly.
	Env []string

	// Files lists host descriptors that become 0, 1, 2, ... in the child:
	// child descriptor i is a duplicate of Files[i]. -1 closes descriptor i.
	// Every other descriptor is closed in the child. At most 16 entries.
	Files []int

	// Setsid puts the child in a new session (it becomes session and process
	// group leader).
	Setsid bool

	// Setctty makes child descriptor number Ctty (an index into Files) the
	// child's controlling terminal. It needs Setsid.
	Setctty bool
	Ctty    int
}

// SpawnError says which step in the child failed and why. A failure of the
// fork itself is reported as a plain Errno instead.
type SpawnError struct {
	Op  string
	Err Errno
}

func (e *SpawnError) Error() string { return "winescape: spawn: " + e.Op + ": " + e.Err.Error() }
func (e *SpawnError) Unwrap() error { return e.Err }

// The steps a child can fail in, reported over the error pipe.
const (
	stepFiles   = 1
	stepSetsid  = 2
	stepSetctty = 3
	stepChdir   = 4
	stepExecve  = 5
)

var spawnStepNames = [...]string{"", "set up descriptors", "setsid", "set controlling terminal", "chdir", "execve"}

// childArgs is everything the child needs, laid out before the fork. It holds
// addresses as uintptr on purpose: the child must not perform pointer writes
// or bounds-checked slice indexing, and everything it reads stays reachable
// from Spawn until after the fork returns.
type childArgs struct {
	path, argv, envp uintptr // NUL-terminated string and pointer arrays
	dir              uintptr // 0 = leave the working directory alone
	errFd            int     // write end of the error pipe, above every fd the child builds
	nfds             int
	setsid           bool
	setctty          bool
	ctty             int
	sigact           [4]uintptr // an all-zero kernel sigaction: SIG_DFL, no flags, empty mask
	sigset           uint64     // the empty signal set
	fds              [maxSpawnFiles]int
}

// Spawn starts path with argv and returns the child's pid. The child is not
// waited for; use Wait4.
func Spawn(path string, argv []string, attr *SpawnAttr) (pid int, err error) {
	if attr == nil {
		attr = &SpawnAttr{}
	}
	if len(attr.Files) > maxSpawnFiles {
		return -1, EINVAL
	}
	if attr.Setctty && (!attr.Setsid || attr.Ctty < 0 || attr.Ctty >= len(attr.Files)) {
		return -1, EINVAL
	}
	if sysClone == 0 || sysExitGroup == 0 || sysRtSigaction == 0 || sysRtSigprocmask == 0 {
		return -1, ErrUnavailable
	}
	if !Available() {
		return -1, ErrUnavailable
	}

	pathP, err := BytePtrFromString(ToUnixPath(path))
	if err != nil {
		return -1, err
	}
	argvPtrs, argvKeep, err := cStringArray(argv)
	if err != nil {
		return -1, err
	}
	envpPtrs, envpKeep, err := cStringArray(attr.Env)
	if err != nil {
		return -1, err
	}
	var dirP *byte
	if attr.Dir != "" {
		if dirP, err = BytePtrFromString(ToUnixPath(attr.Dir)); err != nil {
			return -1, err
		}
	}

	// The child reports a failure between fork and exec over this pipe; a
	// successful execve closes it (close-on-exec), which the parent sees as
	// end of file. The write end is moved above every descriptor the child
	// will build so that building them cannot overwrite it.
	r, w, err := Pipe2(O_CLOEXEC)
	if err != nil {
		return -1, err
	}
	errFd, err := fcntlDupFd(w, maxSpawnFiles*2)
	_ = Close(w)
	if err != nil {
		_ = Close(r)
		return -1, err
	}

	a := new(childArgs)
	a.path = uintptr(unsafe.Pointer(pathP))
	a.argv = uintptr(unsafe.Pointer(&argvPtrs[0]))
	a.envp = uintptr(unsafe.Pointer(&envpPtrs[0]))
	if dirP != nil {
		a.dir = uintptr(unsafe.Pointer(dirP))
	}
	a.errFd = errFd
	a.nfds = len(attr.Files)
	a.setsid = attr.Setsid
	a.setctty = attr.Setctty
	a.ctty = attr.Ctty
	copy(a.fds[:], attr.Files)

	pid, ferr := forkChild(a)

	// Everything the child read is kept alive up to here.
	runtime.KeepAlive(a)
	runtime.KeepAlive(pathP)
	runtime.KeepAlive(argvKeep)
	runtime.KeepAlive(argvPtrs)
	runtime.KeepAlive(envpKeep)
	runtime.KeepAlive(envpPtrs)
	runtime.KeepAlive(dirP)

	_ = Close(errFd)
	if ferr != nil {
		_ = Close(r)
		return -1, ferr
	}

	var report [2]uintptr
	reportBytes := unsafe.Slice((*byte)(unsafe.Pointer(&report[0])), int(unsafe.Sizeof(report)))
	n, rerr := Read(r, reportBytes)
	_ = Close(r)
	if rerr == nil && n == len(reportBytes) {
		// The child got as far as exec and failed; it has already exited.
		var st int32
		_, _ = Wait4(pid, &st, 0, nil)
		step := int(report[0])
		if step < 1 || step >= len(spawnStepNames) {
			step = 0
		}
		return -1, &SpawnError{Op: spawnStepNames[step], Err: Errno(report[1])}
	}
	return pid, nil
}

// cStringArray builds the NULL-terminated pointer array execve wants. The
// second result keeps the strings' storage reachable for the caller. The array
// always has at least its terminating entry, so &array[0] is valid for an empty
// list.
func cStringArray(strs []string) (ptrs []uintptr, keep []*byte, err error) {
	ptrs = make([]uintptr, len(strs)+1)
	keep = make([]*byte, len(strs))
	for i, s := range strs {
		p, perr := BytePtrFromString(s)
		if perr != nil {
			return nil, nil, perr
		}
		keep[i] = p
		ptrs[i] = uintptr(unsafe.Pointer(p))
	}
	return ptrs, keep, nil
}

// fcntlDupFd duplicates fd to the lowest free descriptor >= min with
// close-on-exec set.
func fcntlDupFd(fd, min int) (int, error) {
	r1, _, err := Syscall(sysFcntl, uintptr(fd), fDupFdCloexc, uintptr(min))
	if err != nil {
		return -1, err
	}
	return int(r1), nil
}

// forkChild forks and runs the child's side. It returns in the parent only.
func forkChild(a *childArgs) (int, error) {
	// The signal mask belongs to a thread. Blocking every signal before the
	// fork and restoring the mask after has to happen on one and the same
	// thread, so the goroutine is pinned for the duration.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	all := ^uint64(0)
	var old uint64
	pid, errno := forkAndRun(a, &all, &old)
	if errno != 0 {
		return -1, Errno(errno)
	}
	return int(pid), nil
}

// forkAndRun blocks all signals, forks, and in the child runs spawnChildMain
// (which never returns). In the parent it restores the mask and returns the
// child's pid.
//
// Blocking signals keeps a handler -- Wine's or the Go runtime's -- from running
// in the child's half-copied process before it has exec'd. The child then
// installs an empty mask itself, because exec preserves the mask and the shell
// must not inherit a blocked one.
//
// This is the code that runs on both sides of the fork, so it is nosplit and
// norace like the child's: after the fork returns 0 there is no runtime to
// call into.
//
//go:nosplit
//go:norace
func forkAndRun(a *childArgs, all, old *uint64) (pid uintptr, errno uintptr) {
	_, _, e := syscall6_raw(sysRtSigprocmask, sigSetMask, uintptr(unsafe.Pointer(all)), uintptr(unsafe.Pointer(old)), 8, 0, 0)
	if e != 0 {
		return 0, e
	}
	// clone(SIGCHLD) with no other flags is fork(2); arm64 has no fork.
	r1, _, e := syscall6_raw(sysClone, uintptr(SIGCHLD), 0, 0, 0, 0, 0)
	if e != 0 {
		syscall6_raw(sysRtSigprocmask, sigSetMask, uintptr(unsafe.Pointer(old)), 0, 8, 0, 0)
		return 0, e
	}
	if r1 == 0 {
		spawnChildMain(a)
	}
	syscall6_raw(sysRtSigprocmask, sigSetMask, uintptr(unsafe.Pointer(old)), 0, 8, 0, 0)
	return r1, 0
}

// spawnChildMain is the child after fork. It never returns: it either execs or
// tells the parent why it could not and exits.
//
//go:nosplit
//go:norace
func spawnChildMain(a *childArgs) {
	step, errno := spawnChildRun(a)
	report := [2]uintptr{step, errno}
	syscall6_raw(sysWrite, uintptr(a.errFd), uintptr(unsafe.Pointer(&report[0])), unsafe.Sizeof(report), 0, 0, 0)
	for {
		syscall6_raw(sysExitGroup, 253, 0, 0, 0, 0, 0)
	}
}

// spawnChildRun does the child's work and returns only on failure, with the
// step that failed and its errno. Success is an execve that does not return.
//
//go:nosplit
//go:norace
func spawnChildRun(a *childArgs) (step, errno uintptr) {
	n := a.nfds

	// Rebuild the descriptor table. A source descriptor numbered below its own
	// target would be overwritten by an earlier dup3, so first move those out
	// of the way to numbers above everything in play.
	next := n
	if a.errFd >= next {
		next = a.errFd + 1
	}
	for i := 0; i < n; i++ {
		if a.fds[i] >= next {
			next = a.fds[i] + 1
		}
	}
	for i := 0; i < n; i++ {
		fd := a.fds[i]
		if fd >= 0 && fd < i {
			_, _, e := syscall6_raw(sysDup3, uintptr(fd), uintptr(next), 0, 0, 0, 0)
			if e != 0 {
				return stepFiles, e
			}
			a.fds[i] = next
			next++
		}
	}
	for i := 0; i < n; i++ {
		fd := a.fds[i]
		if fd < 0 {
			syscall6_raw(sysClose, uintptr(i), 0, 0, 0, 0, 0)
			continue
		}
		if fd == i {
			// dup3 refuses oldfd == newfd; what is needed is to clear
			// close-on-exec so the descriptor survives.
			_, _, e := syscall6_raw(sysFcntl, uintptr(fd), fSetFd, 0, 0, 0, 0)
			if e != 0 {
				return stepFiles, e
			}
			continue
		}
		_, _, e := syscall6_raw(sysDup3, uintptr(fd), uintptr(i), 0, 0, 0, 0)
		if e != 0 {
			return stepFiles, e
		}
	}

	// Close everything else, except the error pipe, which the exec closes.
	// close_range needs Linux 5.9; where it is missing the remaining
	// descriptors are left to their own close-on-exec flags.
	if sysCloseRange != 0 {
		if a.errFd > n {
			syscall6_raw(sysCloseRange, uintptr(n), uintptr(a.errFd-1), 0, 0, 0, 0)
		}
		syscall6_raw(sysCloseRange, uintptr(a.errFd+1), 0xffffffff, 0, 0, 0, 0)
	}

	if a.setsid {
		_, _, e := syscall6_raw(sysSetsid, 0, 0, 0, 0, 0, 0)
		if e != 0 {
			return stepSetsid, e
		}
	}
	if a.setctty {
		_, _, e := syscall6_raw(sysIoctl, uintptr(a.ctty), TIOCSCTTY, 0, 0, 0, 0)
		if e != 0 {
			return stepSetctty, e
		}
	}
	if a.dir != 0 {
		_, _, e := syscall6_raw(sysChdir, a.dir, 0, 0, 0, 0, 0)
		if e != 0 {
			return stepChdir, e
		}
	}

	// A signal that was ignored stays ignored across exec, and Wine ignores
	// SIGPIPE in its processes: a shell that inherited that would not die on a
	// broken pipe. Give every signal its default action back.
	for sig := uintptr(1); sig <= lastSignal; sig++ {
		if sig == sigKill || sig == sigStop {
			continue
		}
		syscall6_raw(sysRtSigaction, sig, uintptr(unsafe.Pointer(&a.sigact[0])), 0, 8, 0, 0)
	}
	syscall6_raw(sysRtSigprocmask, sigSetMask, uintptr(unsafe.Pointer(&a.sigset)), 0, 8, 0, 0)

	_, _, e := syscall6_raw(sysExecve, a.path, a.argv, a.envp, 0, 0, 0)
	return stepExecve, e
}

// WaitStatus is the status word Wait4 stores, decoded the way Linux encodes it.
type WaitStatus int32

// Exited reports whether the process ended by returning from main or calling exit.
func (w WaitStatus) Exited() bool { return w&0x7f == 0 }

// ExitStatus is the exit code, meaningful only when Exited.
func (w WaitStatus) ExitStatus() int { return int(w>>8) & 0xff }

// Signaled reports whether a signal ended the process.
func (w WaitStatus) Signaled() bool { return w&0x7f != 0 && w&0x7f != 0x7f }

// Signal is the number of the signal that ended the process, meaningful only
// when Signaled.
func (w WaitStatus) Signal() int { return int(w & 0x7f) }
