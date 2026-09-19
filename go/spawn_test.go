package winescape

import (
	"errors"
	"syscall"
	"testing"
)

// Spawn checks its arguments before it looks at the host, so these hold on any
// machine.
func TestSpawnRejectsBadAttributes(t *testing.T) {
	tests := []struct {
		name string
		attr *SpawnAttr
	}{
		{"too many files", &SpawnAttr{Files: make([]int, maxSpawnFiles+1)}},
		{"setctty without setsid", &SpawnAttr{Files: []int{0}, Setctty: true, Ctty: 0}},
		{"setctty index past the files", &SpawnAttr{Files: []int{0}, Setsid: true, Setctty: true, Ctty: 1}},
		{"setctty negative index", &SpawnAttr{Files: []int{0}, Setsid: true, Setctty: true, Ctty: -1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pid, err := Spawn("/bin/true", []string{"true"}, tc.attr)
			if err != EINVAL || pid != -1 {
				t.Fatalf("Spawn = (%d, %v), want (-1, EINVAL)", pid, err)
			}
		})
	}
}

// Off Wine the raw-syscall path is not there: Spawn must say so and start
// nothing, not fork a real process on a host that merely has the numbers.
func TestSpawnUnavailableOffWine(t *testing.T) {
	if Available() {
		t.Skip("running under Wine")
	}
	pid, err := Spawn("/bin/true", []string{"true"}, nil)
	if err == nil || pid != -1 {
		t.Fatalf("Spawn off Wine = (%d, %v), want (-1, error)", pid, err)
	}
}

func TestCStringArray(t *testing.T) {
	t.Run("empty list still has its terminator", func(t *testing.T) {
		ptrs, keep, err := cStringArray(nil)
		if err != nil || len(ptrs) != 1 || ptrs[0] != 0 || len(keep) != 0 {
			t.Fatalf("cStringArray(nil) = %v, %v, %v", ptrs, keep, err)
		}
	})
	t.Run("entries are NUL-terminated and the list ends in NULL", func(t *testing.T) {
		ptrs, keep, err := cStringArray([]string{"a", "bc"})
		if err != nil || len(ptrs) != 3 || ptrs[2] != 0 || ptrs[0] == 0 || ptrs[1] == 0 {
			t.Fatalf("cStringArray = %v, %v", ptrs, err)
		}
		if len(keep) != 2 || keep[1] == nil || *keep[1] != 'b' {
			t.Fatalf("keep = %v", keep)
		}
	})
	t.Run("an embedded NUL is refused", func(t *testing.T) {
		if _, _, err := cStringArray([]string{"a\x00b"}); err == nil {
			t.Fatal("no error for a string with a NUL byte")
		}
	})
}

func TestWaitStatus(t *testing.T) {
	tests := []struct {
		name     string
		raw      int32
		exited   bool
		code     int
		signaled bool
		signal   int
	}{
		{"exit 0", 0x0000, true, 0, false, 0},
		{"exit 3", 0x0300, true, 3, false, 0},
		{"killed by SIGINT", 0x0002, false, 0, true, 2},
		{"killed by SIGKILL", 0x0009, false, 0, true, 9},
		{"stopped is neither", 0x137f, false, 0, false, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := WaitStatus(tc.raw)
			if w.Exited() != tc.exited || w.Signaled() != tc.signaled {
				t.Fatalf("Exited=%v Signaled=%v, want %v %v", w.Exited(), w.Signaled(), tc.exited, tc.signaled)
			}
			if tc.exited && w.ExitStatus() != tc.code {
				t.Errorf("ExitStatus = %d, want %d", w.ExitStatus(), tc.code)
			}
			if tc.signaled && w.Signal() != tc.signal {
				t.Errorf("Signal = %d, want %d", w.Signal(), tc.signal)
			}
		})
	}
}

// errors.Is must see through SpawnError to the errno, the way it does for the
// library's other errors.
func TestSpawnErrorUnwrapsToErrno(t *testing.T) {
	err := error(&SpawnError{Op: "execve", Err: ENOENT})
	if !errors.Is(err, syscall.ENOENT) || !errors.Is(err, ENOENT) {
		t.Fatalf("errors.Is on %v did not find ENOENT", err)
	}
	var se *SpawnError
	if !errors.As(err, &se) || se.Op != "execve" {
		t.Fatalf("errors.As did not recover the SpawnError: %v", se)
	}
}
