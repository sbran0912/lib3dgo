.PHONY: all build vet test run-vorlage run-vehic run-ray run-boids clean

all: build

build:
	go build -o bin/vorlage ./apps/vorlage
	go build -o bin/vehic ./apps/vehic
	go build -o bin/ray ./apps/ray
	go build -o bin/boids ./apps/boids

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

run-boids:
	go run ./apps/boids

clean:
	rm -rf bin
