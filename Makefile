.POSIX:

.PHONY: test test-wine probe gen clean

test:
	go test -v ./...

gen:
	go run ./cmd/gen-numbers .
	gofmt -w go/numbers_*.go

probe:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o probe/wine_syscall_probe.exe ./probe

test-wine:
	@mkdir -p build
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o build/test_runner.exe ./cmd/test-runner
	wine build/test_runner.exe

clean:
	rm -rf build probe/wine_syscall_probe.exe
