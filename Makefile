.PHONY: test test-race cover vet fmt lint tidy check

test:
	go test ./...

test-race:
	go test -race ./...

cover:
	go test -race -coverprofile=coverage.out -covermode=atomic .
	go tool cover -func=coverage.out

vet:
	go vet ./...

fmt:
	gofmt -w .

lint:
	golangci-lint run

tidy:
	go mod tidy

check: tidy fmt vet test-race lint
