package spec

type TargetOS string
type TargetArch string

const (
	OSLinux   TargetOS = "linux"
	OSFreeBSD TargetOS = "freebsd"

	ArchAMD64   TargetArch = "amd64"
	ArchARM64   TargetArch = "arm64"
	ArchRISCV64 TargetArch = "riscv64"
)

type SyscallEntry struct {
	Name    string
	Args    int
	Numbers map[TargetOS]map[TargetArch]uint64
	Note    string
}

// SyscallTable is the single source of truth for raw syscall numbers.
// Numbers are verified against kernel headers (Linux asm/unistd_64.h,
// asm-generic/unistd.h, FreeBSD sys/syscall.h).
var SyscallTable = []SyscallEntry{
	{
		Name: "read",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 0, ArchARM64: 63},
			OSFreeBSD: {ArchAMD64: 3, ArchARM64: 3},
		},
	},
	{
		Name: "write",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 1, ArchARM64: 64},
			OSFreeBSD: {ArchAMD64: 4, ArchARM64: 4},
		},
	},
	{
		Name: "open",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 2},
			OSFreeBSD: {ArchAMD64: 5, ArchARM64: 5},
		},
		Note: "Linux arm64 uses openat (56)",
	},
	{
		Name: "close",
		Args: 1,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 3, ArchARM64: 57},
			OSFreeBSD: {ArchAMD64: 6, ArchARM64: 6},
		},
	},
	{
		Name: "fstat",
		Args: 2,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 5, ArchARM64: 80},
			OSFreeBSD: {ArchAMD64: 551, ArchARM64: 551},
		},
	},
	{
		Name: "lseek",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 8, ArchARM64: 62},
			OSFreeBSD: {ArchAMD64: 478, ArchARM64: 478},
		},
	},
	{
		Name: "pread64",
		Args: 4,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 17, ArchARM64: 67},
			OSFreeBSD: {ArchAMD64: 475, ArchARM64: 475},
		},
	},
	{
		Name: "pwrite64",
		Args: 4,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 18, ArchARM64: 68},
			OSFreeBSD: {ArchAMD64: 476, ArchARM64: 476},
		},
	},
	{
		Name: "access",
		Args: 2,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 21},
			OSFreeBSD: {ArchAMD64: 33, ArchARM64: 33},
		},
		Note: "Linux arm64 uses faccessat (48)",
	},
	{
		Name: "getpid",
		Args: 0,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 39, ArchARM64: 172},
			OSFreeBSD: {ArchAMD64: 20, ArchARM64: 20},
		},
	},
	{
		Name: "mkdir",
		Args: 2,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 83},
			OSFreeBSD: {ArchAMD64: 136, ArchARM64: 136},
		},
		Note: "Linux arm64 uses mkdirat (34)",
	},
	{
		Name: "rmdir",
		Args: 1,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 84},
			OSFreeBSD: {ArchAMD64: 137, ArchARM64: 137},
		},
		Note: "Linux arm64 uses unlinkat with AT_REMOVEDIR (35)",
	},
	{
		Name: "rename",
		Args: 2,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 82},
			OSFreeBSD: {ArchAMD64: 128, ArchARM64: 128},
		},
		Note: "Linux arm64 uses renameat (38)",
	},
	{
		Name: "unlink",
		Args: 1,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 87},
			OSFreeBSD: {ArchAMD64: 10, ArchARM64: 10},
		},
		Note: "Linux arm64 uses unlinkat (35)",
	},
	{
		Name: "symlink",
		Args: 2,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 88},
			OSFreeBSD: {ArchAMD64: 57, ArchARM64: 57},
		},
		Note: "Linux arm64 uses symlinkat (36)",
	},
	{
		Name: "readlink",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 89},
			OSFreeBSD: {ArchAMD64: 58, ArchARM64: 58},
		},
		Note: "Linux arm64 uses readlinkat (78)",
	},
	{
		Name: "getdents64",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux: {ArchAMD64: 217, ArchARM64: 61},
		},
		Note: "FreeBSD uses getdirentries (554)",
	},
	{
		Name: "getdirentries",
		Args: 4,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSFreeBSD: {ArchAMD64: 554, ArchARM64: 554},
		},
	},
	{
		Name: "openat",
		Args: 4,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 257, ArchARM64: 56},
			OSFreeBSD: {ArchAMD64: 499, ArchARM64: 499},
		},
	},
	{
		Name: "mkdirat",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 258, ArchARM64: 34},
			OSFreeBSD: {ArchAMD64: 500, ArchARM64: 500},
		},
	},
	{
		Name: "newfstatat",
		Args: 4,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux: {ArchAMD64: 262, ArchARM64: 79},
		},
		Note: "FreeBSD uses fstatat (552)",
	},
	{
		Name: "fstatat",
		Args: 4,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSFreeBSD: {ArchAMD64: 552, ArchARM64: 552},
		},
	},
	{
		Name: "unlinkat",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 263, ArchARM64: 35},
			OSFreeBSD: {ArchAMD64: 503, ArchARM64: 503},
		},
	},
	{
		Name: "renameat",
		Args: 4,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 264, ArchARM64: 38},
			OSFreeBSD: {ArchAMD64: 501, ArchARM64: 501},
		},
	},
	{
		// Linux only: RENAME_NOREPLACE (and RENAME_EXCHANGE) have no FreeBSD
		// equivalent syscall -- FreeBSD's renameat(2) has no flags argument,
		// so there is no number to add there without inventing one. Verified
		// against Linux kernel uapi headers (arch/x86/include/generated/uapi/
		// asm/unistd_64.h, include/uapi/asm-generic/unistd.h for arm64), not
		// assumed from renameat's neighboring number.
		Name: "renameat2",
		Args: 5,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux: {ArchAMD64: 316, ArchARM64: 276},
		},
	},
	{
		Name: "readlinkat",
		Args: 4,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 267, ArchARM64: 78},
			OSFreeBSD: {ArchAMD64: 505, ArchARM64: 505},
		},
	},
	{
		Name: "faccessat",
		Args: 4,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 269, ArchARM64: 48, ArchRISCV64: 48},
			OSFreeBSD: {ArchAMD64: 489, ArchARM64: 489},
		},
	},
	{
		Name: "pipe2",
		Args: 2,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 293, ArchARM64: 59, ArchRISCV64: 59},
			OSFreeBSD: {ArchAMD64: 538, ArchARM64: 538},
		},
	},
	{
		Name: "dup3",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 292, ArchARM64: 24, ArchRISCV64: 24},
			OSFreeBSD: {ArchAMD64: 545, ArchARM64: 545},
		},
	},
	{
		Name: "mmap",
		Args: 6,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 9, ArchARM64: 222, ArchRISCV64: 222},
			OSFreeBSD: {ArchAMD64: 477, ArchARM64: 477},
		},
	},
	{
		Name: "munmap",
		Args: 2,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 11, ArchARM64: 215, ArchRISCV64: 215},
			OSFreeBSD: {ArchAMD64: 73, ArchARM64: 73},
		},
	},
	{
		Name: "inotify_init1",
		Args: 1,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux: {ArchAMD64: 294, ArchARM64: 26, ArchRISCV64: 26},
		},
	},
	{
		Name: "inotify_add_watch",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux: {ArchAMD64: 254, ArchARM64: 27, ArchRISCV64: 27},
		},
	},
	{
		Name: "inotify_rm_watch",
		Args: 2,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux: {ArchAMD64: 255, ArchARM64: 28, ArchRISCV64: 28},
		},
	},
	{
		Name: "getuid",
		Args: 0,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 102, ArchARM64: 174, ArchRISCV64: 174},
			OSFreeBSD: {ArchAMD64: 24, ArchARM64: 24},
		},
	},
	{
		Name: "getgid",
		Args: 0,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 104, ArchARM64: 176, ArchRISCV64: 176},
			OSFreeBSD: {ArchAMD64: 47, ArchARM64: 47},
		},
	},
	{
		Name: "geteuid",
		Args: 0,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 107, ArchARM64: 175, ArchRISCV64: 175},
			OSFreeBSD: {ArchAMD64: 25, ArchARM64: 25},
		},
	},
	{
		Name: "getegid",
		Args: 0,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 108, ArchARM64: 177, ArchRISCV64: 177},
			OSFreeBSD: {ArchAMD64: 43, ArchARM64: 43},
		},
	},
	{
		Name: "getppid",
		Args: 0,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 110, ArchARM64: 173, ArchRISCV64: 173},
			OSFreeBSD: {ArchAMD64: 39, ArchARM64: 39},
		},
	},
	{
		Name: "socket",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 41, ArchARM64: 198, ArchRISCV64: 198},
			OSFreeBSD: {ArchAMD64: 97, ArchARM64: 97},
		},
	},
	{
		Name: "connect",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 42, ArchARM64: 203, ArchRISCV64: 203},
			OSFreeBSD: {ArchAMD64: 98, ArchARM64: 98},
		},
	},
	{
		Name: "bind",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 49, ArchARM64: 200, ArchRISCV64: 200},
			OSFreeBSD: {ArchAMD64: 104, ArchARM64: 104},
		},
	},
	{
		Name: "listen",
		Args: 2,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 50, ArchARM64: 201, ArchRISCV64: 201},
			OSFreeBSD: {ArchAMD64: 106, ArchARM64: 106},
		},
	},
	{
		Name: "accept4",
		Args: 4,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 288, ArchARM64: 242, ArchRISCV64: 242},
			OSFreeBSD: {ArchAMD64: 541, ArchARM64: 541},
		},
	},
	{
		Name: "ioctl",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 16, ArchARM64: 29, ArchRISCV64: 29},
			OSFreeBSD: {ArchAMD64: 54, ArchARM64: 54},
		},
	},
	{
		Name: "nanosleep",
		Args: 2,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 35, ArchARM64: 101, ArchRISCV64: 101},
			OSFreeBSD: {ArchAMD64: 240, ArchARM64: 240},
		},
	},
	{
		Name: "clock_gettime",
		Args: 2,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 228, ArchARM64: 113, ArchRISCV64: 113},
			OSFreeBSD: {ArchAMD64: 232, ArchARM64: 232},
		},
	},
	{
		Name: "clock_nanosleep",
		Args: 4,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 230, ArchARM64: 115, ArchRISCV64: 115},
			OSFreeBSD: {ArchAMD64: 244, ArchARM64: 244},
		},
	},
	{
		Name: "kill",
		Args: 2,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 62, ArchARM64: 129, ArchRISCV64: 129},
			OSFreeBSD: {ArchAMD64: 37, ArchARM64: 37},
		},
	},
	{
		Name: "wait4",
		Args: 4,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 61, ArchARM64: 260, ArchRISCV64: 260},
			OSFreeBSD: {ArchAMD64: 7, ArchARM64: 7},
		},
	},
	{
		Name: "fchmodat",
		Args: 4,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 268, ArchARM64: 53, ArchRISCV64: 53},
			OSFreeBSD: {ArchAMD64: 490, ArchARM64: 490},
		},
	},
	{
		Name: "fchownat",
		Args: 5,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 260, ArchARM64: 54, ArchRISCV64: 54},
			OSFreeBSD: {ArchAMD64: 491, ArchARM64: 491},
		},
	},
	{
		Name: "utimensat",
		Args: 4,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 280, ArchARM64: 88, ArchRISCV64: 88},
			OSFreeBSD: {ArchAMD64: 547, ArchARM64: 547},
		},
	},
	{
		Name: "ftruncate",
		Args: 2,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 77, ArchARM64: 46, ArchRISCV64: 46},
			OSFreeBSD: {ArchAMD64: 480, ArchARM64: 480},
		},
	},
	{
		Name: "symlinkat",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 266, ArchARM64: 36, ArchRISCV64: 36},
			OSFreeBSD: {ArchAMD64: 504, ArchARM64: 504},
		},
	},
	{
		Name: "copy_file_range",
		Args: 6,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 326, ArchARM64: 285, ArchRISCV64: 285},
			OSFreeBSD: {ArchAMD64: 569, ArchARM64: 569},
		},
	},
	{
		Name: "flock",
		Args: 2,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 73, ArchARM64: 32, ArchRISCV64: 32},
			OSFreeBSD: {ArchAMD64: 131, ArchARM64: 131},
		},
	},
	{
		Name: "statfs",
		Args: 2,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 137, ArchARM64: 43, ArchRISCV64: 43},
			OSFreeBSD: {ArchAMD64: 396, ArchARM64: 396},
		},
	},
	{
		Name: "fstatfs",
		Args: 2,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 138, ArchARM64: 44, ArchRISCV64: 44},
			OSFreeBSD: {ArchAMD64: 397, ArchARM64: 397},
		},
	},
	{
		Name: "execve",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux:   {ArchAMD64: 59, ArchARM64: 221, ArchRISCV64: 221},
			OSFreeBSD: {ArchAMD64: 59, ArchARM64: 59},
		},
	},
	{
		Name: "clone",
		Args: 5,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux: {ArchAMD64: 56, ArchARM64: 220, ArchRISCV64: 220},
		},
	},
	{
		Name: "setsid",
		Args: 0,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux: {ArchAMD64: 112, ArchARM64: 157, ArchRISCV64: 157},
		},
	},
	{
		Name: "chdir",
		Args: 1,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux: {ArchAMD64: 80, ArchARM64: 49, ArchRISCV64: 49},
		},
	},
	{
		Name: "exit_group",
		Args: 1,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux: {ArchAMD64: 231, ArchARM64: 94, ArchRISCV64: 94},
		},
	},
	{
		Name: "rt_sigprocmask",
		Args: 4,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux: {ArchAMD64: 14, ArchARM64: 135, ArchRISCV64: 135},
		},
	},
	{
		Name: "rt_sigaction",
		Args: 4,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux: {ArchAMD64: 13, ArchARM64: 134, ArchRISCV64: 134},
		},
	},
	{
		Name: "ppoll",
		Args: 5,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux: {ArchAMD64: 271, ArchARM64: 73, ArchRISCV64: 73},
		},
	},
	{
		Name: "close_range",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux: {ArchAMD64: 436, ArchARM64: 436, ArchRISCV64: 436},
		},
	},
	{
		Name: "fcntl",
		Args: 3,
		Numbers: map[TargetOS]map[TargetArch]uint64{
			OSLinux: {ArchAMD64: 72, ArchARM64: 25, ArchRISCV64: 25},
		},
	},
}
