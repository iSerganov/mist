test:
	go test -race -count=1 ./...

lint:
	golangci-lint run --timeout=5m

.PHONY: test lint
