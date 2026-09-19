# Go Runtime & Threading Considerations

## The Challenge

Standard Go `syscall` invocations are wrapped by runtime hooks (`entersyscall` / `exitsyscall`) which release the logical processor (`P`) so other goroutines can continue execution while an OS thread is blocked.

Because `libwinescape` invokes raw assembly `SYSCALL` instructions without cgo and without private runtime hooks, the Go scheduler is unaware that the OS thread has entered the kernel. If a blocking syscall (such as reading from a pipe or slow socket) is issued directly on an arbitrary goroutine, the OS thread and its associated `P` remain blocked.

Key implications:
1. **GOMAXPROCS Sizing:** For applications doing concurrent raw blocking I/O, `GOMAXPROCS` should be configured to ensure sufficient spare `P` processors remain available for other runnable goroutines.
2. **Stop-The-World Awareness:** Avoid triggering global `stopTheWorld` operations (e.g. `runtime.ReadMemStats` or heavy allocations that force STW GC) while waiting on inter-goroutine unblocking rendezvous across raw syscalls.

## The Solution: `winescape/gort`

For non-blocking filesystem I/O on local files, direct calls have minimal latency.

For operations that may block or when strict scheduler isolation is required, `libwinescape` provides the `gort` subpackage:
- A worker pool of goroutines bound to OS threads via `runtime.LockOSThread()`.
- Explicit bounded concurrency.
- Pure Go implementation without cgo.

## Usage Example

```go
package main

import (
	"fmt"
	"log"

	"github.com/unxed/libwinescape/go"
	"github.com/unxed/libwinescape/go/gort"
)

func main() {
	// Create a worker pool with 4 OS-thread-locked workers
	pool := gort.NewPool(gort.WithWorkers(4))
	defer pool.Close()

	// Execute a filesystem call safely inside the worker pool
	fd, err := gort.RunInPool(pool, func() (int, error) {
		return winescape.Open("/etc/hosts", winescape.O_RDONLY, 0)
	})
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer winescape.Close(fd)

	buf := make([]byte, 128)
	n, err := gort.RunInPool(pool, func() (int, error) {
		return winescape.Read(fd, buf)
	})
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	fmt.Printf("Read %d bytes: %s\n", n, string(buf[:n]))
}
```

## Spawning a process (`Spawn`, `StartPTY`)

`Spawn` forks the Wine process. Only the forking thread exists in the child; every
other thread, and every lock one of them held, is a frozen copy. Two rules follow,
and the implementation keeps both (`go/spawn.go`):

1. **The child runs raw syscalls only**, from `nosplit` code, on data laid out
   before the fork: no allocation, no stack growth, no Go runtime and no Wine call.
   It rebuilds the descriptor table, closes the rest, calls `setsid`, sets the
   controlling terminal, changes directory, gives every signal its default action
   back (Wine ignores `SIGPIPE`, and an ignored signal survives `execve`), clears the
   signal mask and execs. If any step fails it reports the step and errno over a
   close-on-exec pipe and exits; the parent turns that into a `SpawnError`.
2. **Signals are blocked across the fork**, on a goroutine pinned to its thread
   (`runtime.LockOSThread`), so no handler runs in the child before it has exec'd.

`fork` is `clone(SIGCHLD)` because arm64 has no `fork`. A `Spawn` call costs a copy of
the address space's page tables; it is meant for starting a shell, not for a hot loop.

Reading a pseudo-terminal master is a blocking raw read like any other. Either call it
on a `gort` worker, or `Poll` with a short timeout first so the thread is not held while
the child is quiet. `Wait4` with `WNOHANG` is the non-blocking way to reap.
