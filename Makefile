GOCMD=go
GOFLAGS=-trimpath -buildvcs
BINARY=bin/kavos

.PHONY: build run test clean

build:
	$(GOCMD) build $(GOFLAGS) -o $(BINARY) ./cmd/server

run:
	$(GOCMD) run $(GOFLAGS) ./cmd/server

clean:
	rm -rf bin
