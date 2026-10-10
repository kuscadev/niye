.PHONY: build test vet install clean

BINARY=niye
VERSION=0.1.0

build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY) ./cmd/niye

test:
	go test -v -race ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

install: build
	install -m 755 $(BINARY) $(HOME)/.local/bin/$(BINARY)

clean:
	rm -f $(BINARY) ee

check: fmt vet test

.DEFAULT_GOAL := check
