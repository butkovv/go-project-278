build:
	go build -o bin/url-shortener ./cmd/api

lint:
	golangci-lint run ./...

test:
	go mod tidy
	go test -v ./... -race
