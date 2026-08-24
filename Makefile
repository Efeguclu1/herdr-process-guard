.PHONY: build test race vet fmt-check check clean

BINARY := bin/herdr-process-guard

build:
	go build -trimpath -o $(BINARY) ./cmd/herdr-process-guard

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

check: fmt-check vet race build

clean:
	rm -f $(BINARY)
