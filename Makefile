.PHONY: all build vet test run-vorlage run-vehic run-ray clean

all: build

build:
	go build -o bin/vorlage ./apps/vorlage
	go build -o bin/vehic ./apps/vehic
	go build -o bin/ray ./apps/ray

vet:
	go vet ./...

test:
	go test ./...

run-vorlage:
	go run ./apps/vorlage

run-vehic:
	go run ./apps/vehic

run-ray:
	go run ./apps/ray

clean:
	rm -rf bin
