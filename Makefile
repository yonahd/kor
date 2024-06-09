.PHONY: *

build:
	go build -o build/kor main.go

clean:
	rm -fr build coverage.txt coverage.html

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

test:
	go test -race -coverprofile=coverage.txt -shuffle on ./...

cover: test
	go tool cover -func=coverage.txt
	go tool cover -o coverage.html -html=coverage.txt
