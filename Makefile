EXCEPTIONS_DIR := pkg/kor/exceptions
EXCEPTIONS_FILE_PATTERN := *.json

.PHONY: *

build:
	go build -o build/kor main.go

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

test:
	go test -race -coverprofile=coverage.txt -shuffle on ./...

sort-exception-files:
	@echo "Sorting exception files..."
	@find pkg/kor/exceptions -name '*.json' -exec sh -c ' \
		jq "with_entries(.value |= sort_by(.Namespace, .ResourceName))" {} > {}.tmp && mv {}.tmp {} \
	' \;

validate-exception-sorting:
	@$(foreach file, $(wildcard $(EXCEPTIONS_DIR)/*/*.json), \
		$(eval SORTED := $(shell jq "with_entries(.value |= sort_by(.Namespace, .ResourceName))" "$(file)")) \
		$(eval CURRENT_FILE := $(shell jq . $(file))) \
		if [ "$(CURRENT_FILE)" != "$(SORTED)" ]; then \
			echo $(file); \
		fi; \
	)
