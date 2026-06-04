VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
PREFIX ?= $(HOME)/.local

.PHONY: build install test vet fmt check demo clean

build:
	go build -ldflags "$(LDFLAGS)" -o pad .

install:
	go build -ldflags "$(LDFLAGS)" -o $(PREFIX)/bin/pad .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

check: fmt vet test

demo:
	vhs demo.tape

clean:
	rm -f pad
	rm -rf dist
