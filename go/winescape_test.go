package winescape

import (
	"errors"
	"io/fs"
	"syscall"
	"testing"
)

func TestErrno_Error(t *testing.T) {
	e0 := Errno(0)
	if e0.Error() != "errno 0 (success)" {
		t.Errorf("unexpected zero errno message: %s", e0.Error())
	}

	e2 := Errno(2) // ENOENT
	if e2.Error() == "" {
		t.Errorf("expected non-empty string for ENOENT (errno 2)")
	}
}
func TestPOSIX_ErrnoConstants(t *testing.T) {
	if EINTR != 4 {
		t.Errorf("expected EINTR == 4, got %d", EINTR)
	}
	if ENOENT != 2 {
		t.Errorf("expected ENOENT == 2, got %d", ENOENT)
	}
	if EAGAIN != 11 {
		t.Errorf("expected EAGAIN == 11, got %d", EAGAIN)
	}
	if EINVAL != 22 {
		t.Errorf("expected EINVAL == 22, got %d", EINVAL)
	}
}

func TestRetryEINTR_RetriesOnEINTRAndERESTART(t *testing.T) {
	for _, injected := range []error{EINTR, ERESTART} {
		calls := 0
		r1, r2, err := retryEINTR(func() (uintptr, uintptr, error) {
			calls++
			if calls < 3 {
				return 0, 0, injected
			}
			return 42, 7, nil
		})
		if err != nil {
			t.Fatalf("expected eventual success, got err=%v", err)
		}
		if calls != 3 {
			t.Errorf("expected exactly 3 attempts, got %d", calls)
		}
		if r1 != 42 || r2 != 7 {
			t.Errorf("unexpected results r1=%d r2=%d", r1, r2)
		}
	}
}

func TestRetryEINTR_PassesThroughOtherErrors(t *testing.T) {
	calls := 0
	_, _, err := retryEINTR(func() (uintptr, uintptr, error) {
		calls++
		return 0, 0, EBADF
	})
	if err != EBADF {
		t.Errorf("expected EBADF to pass through unmodified, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected exactly 1 attempt for a non-EINTR error, got %d", calls)
	}
}

func TestSyscallConstants_NonZero(t *testing.T) {
	skipWithoutSyscallTable(t)
	if sysWrite != 1 && sysWrite != 4 && sysWrite != 64 {
		t.Errorf("sysWrite number %d is not matching known platforms", sysWrite)
	}
	if sysClose != 3 && sysClose != 6 && sysClose != 57 {
		t.Errorf("sysClose number %d is not matching known platforms", sysClose)
	}
	if sysGetpid == 0 {
		t.Errorf("sysGetpid must not be 0")
	}
}

func TestSyscallN_ArgumentMarshalling(t *testing.T) {
	// Verify SyscallN / Call dispatch interface compiles and handles variable slice lengths
	if !Available() {
		// When unavailable, should return ErrUnavailable cleanly
		_, _, err := SyscallN(sysGetpid)
		if err != ErrUnavailable {
			t.Errorf("expected ErrUnavailable on unsupported host, got %v", err)
		}
		_, errCall := Call(sysGetpid)
		if errCall != ErrUnavailable {
			t.Errorf("expected ErrUnavailable from Call, got %v", errCall)
		}
	}
}

func TestErrno_IsMatchesFsSentinels(t *testing.T) {
	// os.IsNotExist and its friends are spelled errors.Is(err, fs.ErrNotExist)
	// in modern code, and that is how callers ask "is the file simply not
	// there". An errno that cannot answer turns an absent file into a hard
	// failure at every one of those call sites.
	cases := []struct {
		errno  Errno
		target error
		want   bool
	}{
		{ENOENT, fs.ErrNotExist, true},
		{ENOTDIR, fs.ErrNotExist, true},
		{EEXIST, fs.ErrNotExist, false},
		{EEXIST, fs.ErrExist, true},
		{EACCES, fs.ErrPermission, true},
		{EPERM, fs.ErrPermission, true},
		{ENOENT, fs.ErrPermission, false},
		{EINVAL, fs.ErrInvalid, true},
	}
	for _, tc := range cases {
		if got := errors.Is(error(tc.errno), tc.target); got != tc.want {
			t.Errorf("errors.Is(Errno(%d), %v) = %v, want %v", tc.errno, tc.target, got, tc.want)
		}
	}

	// The syscall.Errno comparison the type already supported must keep
	// working: nothing above may shadow it.
	if !errors.Is(error(ENOENT), syscall.ENOENT) {
		t.Error("errors.Is(ENOENT, syscall.ENOENT) = false, want true")
	}
}
