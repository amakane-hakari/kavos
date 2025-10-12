.PHONY: run build report tidy

run:
	go run ./cmd/loadtest $(ARGS)

build:
	go build -o bin/loadtest ./cmd/loadtest

report:
	@cat vegeta_results.bin | vegeta report

tidy:
	go mod tidy