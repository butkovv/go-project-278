build:
	go build -o bin/app ./cmd/api

lint:
	golangci-lint run ./...

test:
	go mod tidy
	go test -v ./... -race
