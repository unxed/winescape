package winescape

import (
	"strconv"
	"time"
	"unsafe"
)

// Pseudo-terminal, foreground-group and poll support: what an embedded terminal
// needs around Spawn (docs/SCOPE.md, the terminal case). Host Linux only, like
// Spawn.

// Linux terminal ioctls, identical on amd64 and arm64.
const (
	TIOCGPGRP  = 0x540F
	TIOCSPGRP  = 0x5410
	TIOCGPTN   = 0x80045430
	TIOCSPTLCK = 0x40045431
)

// Poll events and results (poll(2)).
const (
	POLLIN   = 0x0001
	POLLOUT  = 0x0004
	POLLERR  = 0x0008
	POLLHUP  = 0x0010
	POLLNVAL = 0x0020
)

// PollFd is struct pollfd.
type PollFd struct {
	Fd      int32
	Events  int16
	Revents int16
}

// Poll waits until one of fds is ready or timeout passes; a negative timeout
// waits without limit. It returns the number of entries with a non-zero
// Revents. A wait interrupted by a signal is restarted.
//
// A blocking read from a terminal master holds its OS thread for as long as the
// child is quiet (docs/threading.md). Polling with a short timeout, or reading
// on a gort worker, is what keeps that from costing the Go scheduler a P.
func Poll(fds []PollFd, timeout time.Duration) (int, error) {
	if sysPpoll == 0 {
		return -1, ErrUnavailable
	}
	var base uintptr
	if len(fds) > 0 {
		base = uintptr(unsafe.Pointer(&fds[0]))
	}
	var tsp uintptr
	var ts Timespec
	if timeout >= 0 {
		ts = Timespec{Sec: int64(timeout / time.Second), Nsec: int64(timeout % time.Second)}
		tsp = uintptr(unsafe.Pointer(&ts))
	}
	r1, _, err := retryEINTR(func() (uintptr, uintptr, error) {
		return Syscall6(sysPpoll, base, uintptr(len(fds)), tsp, 0, 8, 0)
	})
	if err != nil {
		return -1, err
	}
	return int(r1), nil
}

// SetWinsize sets the terminal window size of fd. On a pseudo-terminal master it
// resizes the slave, and the foreground process group gets SIGWINCH.
func SetWinsize(fd int, ws *Winsize) error {
	return Ioctl(fd, TIOCSWINSZ, unsafe.Pointer(ws))
}

// Tcgetpgrp returns the foreground process group of the terminal on fd. On a
// pseudo-terminal master it answers for the slave, so comparing it with the
// child's pid says whether the shell is running something in the foreground.
func Tcgetpgrp(fd int) (int, error) {
	var pgrp int32
	if err := Ioctl(fd, TIOCGPGRP, unsafe.Pointer(&pgrp)); err != nil {
		return -1, err
	}
	return int(pgrp), nil
}

// OpenPTY allocates a pseudo-terminal and returns the master, opened
// close-on-exec, and the path of the slave (/dev/pts/N). The slave is unlocked.
func OpenPTY() (master int, slave string, err error) {
	fd, err := Open("/dev/ptmx", O_RDWR|O_NOCTTY|O_CLOEXEC, 0)
	if err != nil {
		return -1, "", err
	}
	var unlock int32
	if err := Ioctl(fd, TIOCSPTLCK, unsafe.Pointer(&unlock)); err != nil {
		_ = Close(fd)
		return -1, "", err
	}
	var n uint32
	if err := Ioctl(fd, TIOCGPTN, unsafe.Pointer(&n)); err != nil {
		_ = Close(fd)
		return -1, "", err
	}
	return fd, "/dev/pts/" + strconv.FormatUint(uint64(n), 10), nil
}

// StartPTY starts path on a new pseudo-terminal, the way forkpty does: the child
// leads a new session, the slave is its controlling terminal and its standard
// input, output and error, and the window size is ws. It returns the master, on
// which the caller reads the child's output and writes its input, and the pid.
//
// Closing the master hangs the child up. The caller reaps it with Wait4.
func StartPTY(path string, argv, env []string, dir string, ws Winsize) (master, pid int, err error) {
	master, slavePath, err := OpenPTY()
	if err != nil {
		return -1, -1, err
	}
	if err := SetWinsize(master, &ws); err != nil {
		_ = Close(master)
		return -1, -1, err
	}
	// Opened without O_NOCTTY the slave could become the controlling terminal
	// of this Wine process; only the child should get it.
	slave, err := Open(slavePath, O_RDWR|O_NOCTTY|O_CLOEXEC, 0)
	if err != nil {
		_ = Close(master)
		return -1, -1, err
	}
	pid, err = Spawn(path, argv, &SpawnAttr{
		Dir:     dir,
		Env:     env,
		Files:   []int{slave, slave, slave},
		Setsid:  true,
		Setctty: true,
		Ctty:    0,
	})
	_ = Close(slave)
	if err != nil {
		_ = Close(master)
		return -1, -1, err
	}
	return master, pid, nil
}
