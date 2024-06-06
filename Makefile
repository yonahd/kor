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
	@$(eval FAILED_FILES := $(shell \
		find $(EXCEPTIONS_DIR) -name '$(EXCEPTIONS_FILE_PATTERN)' -exec sh -c ' \
			SORTED=$$(jq "with_entries(.value |= sort_by(.Namespace, .ResourceName))" "$$1"); \
			if [ "$$(jq . "$$1")" != "$$SORTED" ]; then \
				echo "$$1"; \
			fi \
		' sh {} \; \
	))

	@if [ -z "$(FAILED_FILES)" ]; then \
		echo "All files sorted correctly."; \
	else \
		echo "The following JSON files are not sorted:"; \
		for file in $(FAILED_FILES); do \
			echo "\t$$file"; \
		done \
	fi
