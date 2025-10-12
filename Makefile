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

.PHONY: load-k6 load-vegeta
load_k6:
	BASE_URL?=http://localhost:8080 ; \
	PATH_PATTERN?=/kvs/%s ; \
	DURATION?=1m ; RATE?=500 ; VUS?=100 ; READ_RATIO?=0.9 ; KEYS?=50000 ' VALUE_SIZE?=128 ; TTL_RATIO?=0 ; TTL_MS?=0 ; \
	k6 run scripts/k6/mixed_workload.js

load-vegeta:
	chmod +x scripts/vegeta/mixed.sh
	BASE_URL=$${BASE_URL:-http://localhost:8080} \
	PATH_PATTERN=$${PATH_PATTERN:-/kvs/%s} \
	RATE=$${RATE:-500} DURATION=$${DURATION:-1m} READ_RATIO=$${READ_RATIO:-0.9} \
	KEYS=$${KEYS:-50000} VALUE_SIZE=$${VALUE_SIZE:-128} TTL_RATIO=$${TTL_RATIO:-0} TTL_MS=$${TTL_MS:-0} \
    OUT=$${OUT:-vegeta_mixed.bin} \
	scripts/vegeta/mixed.sh