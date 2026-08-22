.PHONY: run build test fmt vet

run:
	go run cmd/api/main.go

build:
	go build -o bin/api cmd/api/main.go

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...
