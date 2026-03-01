.PHONY: run test

run:
	go run cmd/cli/main.go start --dir ./examples/goback

test:
	go run gotest.tools/gotestsum@v1.13.0 --format pkgname -- -count=1 ./...