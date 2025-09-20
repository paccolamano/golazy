.PHONY: all
all: lint test

.PHONY: lint
lint:
	go tool golangci-lint run

.PHONY: test
test:
	go test -v -cover -parallel 3 ./...
