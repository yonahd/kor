.PHONY: *

build:
	go build -o build/kor main.go

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

test:
	go test -race -coverprofile=coverage.txt -shuffle on ./...
