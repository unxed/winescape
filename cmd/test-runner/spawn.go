package main

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	winescape "github.com/unxed/libwinescape/go"
)

// ptyEnv is what the shells under test get: nothing from the Wine process.
var ptyEnv = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "TERM=xterm", "HOME=/tmp"}

// readPTY collects output from a terminal master until done says it has
// enough, or, with a nil done, until the child hangs up. Linux reports the
// hang-up as EIO on the master.
func readPTY(master int, done func([]byte) bool, limit time.Duration) ([]byte, error) {
	var out []byte
	buf := make([]byte, 4096)
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		fds := []winescape.PollFd{{Fd: int32(master), Events: winescape.POLLIN}}
		n, err := winescape.Poll(fds, 200*time.Millisecond)
		if err != nil {
			return out, fmt.Errorf("poll: %w", err)
		}
		if n == 0 {
			continue
		}
		m, err := winescape.Read(master, buf)
		if m > 0 {
			out = append(out, buf[:m]...)
			if done != nil && done(out) {
				return out, nil
			}
		}
		if err != nil {
			if errors.Is(err, winescape.EIO) {
				return out, nil
			}
			return out, fmt.Errorf("read: %w", err)
		}
		if m == 0 && fds[0].Revents&winescape.POLLHUP != 0 {
			return out, nil
		}
	}
	return out, fmt.Errorf("timed out after %v with output %q", limit, out)
}

// waitPid reaps pid, giving up after limit.
func waitPid(pid int, limit time.Duration) (winescape.WaitStatus, error) {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		var st int32
		got, err := winescape.Wait4(pid, &st, 1 /* WNOHANG */, nil)
		if err != nil {
			return 0, err
		}
		if got == pid {
			return winescape.WaitStatus(st), nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return 0, fmt.Errorf("pid %d did not exit within %v", pid, limit)
}

// runPTY runs a shell command on a fresh pseudo-terminal and returns its
// output and exit status.
func runPTY(script string, ws winescape.Winsize, dir string) (string, winescape.WaitStatus, error) {
	master, pid, err := winescape.StartPTY("/bin/sh", []string{"sh", "-c", script}, ptyEnv, dir, ws)
	if err != nil {
		return "", 0, err
	}
	defer winescape.Close(master)
	out, err := readPTY(master, nil, 15*time.Second)
	if err != nil {
		_ = winescape.Kill(pid, winescape.SIGKILL)
		_, _ = waitPid(pid, 2*time.Second)
		return string(out), 0, err
	}
	st, err := waitPid(pid, 5*time.Second)
	return string(out), st, err
}

var defaultWinsize = winescape.Winsize{Row: 24, Col: 80}

func runSpawnTests(runTest func(string, func() error)) {
	runTest("Spawn: pty output and exit status", func() error {
		out, st, err := runPTY("echo hello-pty; exit 3", defaultWinsize, "/tmp")
		if err != nil {
			return err
		}
		if !strings.Contains(out, "hello-pty") {
			return fmt.Errorf("output %q lacks the greeting", out)
		}
		if !st.Exited() || st.ExitStatus() != 3 {
			return fmt.Errorf("status %#x, want exit 3", int32(st))
		}
		return nil
	})

	runTest("Spawn: the slave is the controlling terminal", func() error {
		out, _, err := runPTY("tty", defaultWinsize, "/tmp")
		if err != nil {
			return err
		}
		if !strings.HasPrefix(strings.TrimSpace(out), "/dev/pts/") {
			return fmt.Errorf("tty printed %q, want /dev/pts/N", out)
		}
		return nil
	})

	runTest("Spawn: SIGPIPE is not left ignored and the signal mask is empty", func() error {
		// Wine ignores SIGPIPE in its own processes; a shell that inherited
		// that would survive a broken pipe.
		out, _, err := runPTY(`while read k v; do case $k in SigIgn:|SigBlk:) echo $k $v;; esac; done < /proc/$$/status`, defaultWinsize, "/tmp")
		if err != nil {
			return err
		}
		vals := map[string]uint64{}
		for _, line := range strings.Split(strings.ReplaceAll(out, "\r", ""), "\n") {
			f := strings.Fields(line)
			if len(f) == 2 {
				n, perr := strconv.ParseUint(f[1], 16, 64)
				if perr == nil {
					vals[f[0]] = n
				}
			}
		}
		ign, okIgn := vals["SigIgn:"]
		blk, okBlk := vals["SigBlk:"]
		if !okIgn || !okBlk {
			return fmt.Errorf("could not read the masks from %q", out)
		}
		if ign&(1<<(13-1)) != 0 {
			return fmt.Errorf("SIGPIPE is ignored in the child (SigIgn %016x)", ign)
		}
		if blk != 0 {
			return fmt.Errorf("signals are blocked in the child (SigBlk %016x)", blk)
		}
		return nil
	})

	runTest("Spawn: initial window size", func() error {
		out, _, err := runPTY("stty size", winescape.Winsize{Row: 30, Col: 100}, "/tmp")
		if err != nil {
			return err
		}
		if strings.TrimSpace(out) != "30 100" {
			return fmt.Errorf("stty size printed %q, want \"30 100\"", out)
		}
		return nil
	})

	runTest("Spawn: working directory", func() error {
		out, _, err := runPTY("pwd", defaultWinsize, "/usr")
		if err != nil {
			return err
		}
		if strings.TrimSpace(out) != "/usr" {
			return fmt.Errorf("pwd printed %q, want /usr", out)
		}
		return nil
	})

	runTest("Spawn: ^C on the master interrupts the foreground group", func() error {
		master, pid, err := winescape.StartPTY("/bin/sh", []string{"sh", "-c", "exec sleep 30"}, ptyEnv, "/tmp", defaultWinsize)
		if err != nil {
			return err
		}
		defer winescape.Close(master)
		time.Sleep(500 * time.Millisecond)
		if _, err := winescape.Write(master, []byte{3}); err != nil {
			return err
		}
		st, err := waitPid(pid, 5*time.Second)
		if err != nil {
			_ = winescape.Kill(pid, winescape.SIGKILL)
			return err
		}
		if !st.Signaled() || st.Signal() != winescape.SIGINT {
			return fmt.Errorf("status %#x, want death by SIGINT", int32(st))
		}
		return nil
	})

	runTest("Spawn: Tcgetpgrp on the master is the session leader's group", func() error {
		master, pid, err := winescape.StartPTY("/bin/sh", []string{"sh", "-c", "exec sleep 30"}, ptyEnv, "/tmp", defaultWinsize)
		if err != nil {
			return err
		}
		defer winescape.Close(master)
		time.Sleep(300 * time.Millisecond)
		pg, err := winescape.Tcgetpgrp(master)
		_ = winescape.Kill(pid, winescape.SIGKILL)
		_, _ = waitPid(pid, 5*time.Second)
		if err != nil {
			return err
		}
		if pg != pid {
			return fmt.Errorf("foreground group %d, want the child's pid %d", pg, pid)
		}
		return nil
	})

	runTest("Spawn: a failed execve is reported, not swallowed", func() error {
		_, err := winescape.Spawn("/nonexistent/definitely-not-here", []string{"x"}, &winescape.SpawnAttr{})
		var se *winescape.SpawnError
		if !errors.As(err, &se) || se.Op != "execve" || !errors.Is(err, winescape.ENOENT) {
			return fmt.Errorf("err = %v, want an execve SpawnError wrapping ENOENT", err)
		}
		return nil
	})

	runTest("Spawn: pipes as stdout and stderr, stdin closed", func() error {
		r, w, err := winescape.Pipe2(winescape.O_CLOEXEC)
		if err != nil {
			return err
		}
		defer winescape.Close(r)
		pid, err := winescape.Spawn("/bin/sh", []string{"sh", "-c", "echo out; echo err 1>&2"},
			&winescape.SpawnAttr{Env: ptyEnv, Files: []int{-1, w, w}})
		_ = winescape.Close(w)
		if err != nil {
			return err
		}
		out, err := readPTY(r, nil, 10*time.Second)
		if err != nil {
			return err
		}
		if _, err := waitPid(pid, 5*time.Second); err != nil {
			return err
		}
		if string(out) != "out\nerr\n" {
			return fmt.Errorf("pipe carried %q, want \"out\\nerr\\n\"", out)
		}
		return nil
	})

	runTest("Spawn: forking while other goroutines allocate", func() error {
		// The child is a copy of a process whose other threads are busy; it
		// must get to exec without touching any of that.
		stop := make(chan struct{})
		defer close(stop)
		for i := 0; i < runtime.NumCPU(); i++ {
			go func() {
				var keep [][]byte
				for {
					select {
					case <-stop:
						return
					default:
					}
					keep = append(keep, make([]byte, 1<<16))
					if len(keep) > 64 {
						keep = nil
						runtime.GC()
					}
				}
			}()
		}
		for i := 0; i < 40; i++ {
			pid, err := winescape.Spawn("/bin/true", []string{"true"}, &winescape.SpawnAttr{Env: ptyEnv})
			if err != nil {
				return fmt.Errorf("spawn %d: %w", i, err)
			}
			st, err := waitPid(pid, 10*time.Second)
			if err != nil {
				return fmt.Errorf("wait %d: %w", i, err)
			}
			if !st.Exited() || st.ExitStatus() != 0 {
				return fmt.Errorf("child %d: status %#x", i, int32(st))
			}
		}
		return nil
	})
}
