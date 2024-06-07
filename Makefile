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
	@find $(EXCEPTIONS_DIR) -name '$(EXCEPTIONS_FILE_PATTERN)' -exec sh -c ' \
		jq "with_entries(.value |= sort_by(.Namespace, .ResourceName))" {} > {}.tmp && mv {}.tmp {} \
	' \;

validate-exception-sorting:
	@PRINT_ERR=1; \
	for file in $(wildcard $(EXCEPTIONS_DIR)/*/$(EXCEPTIONS_FILE_PATTERN)); do \
		SORTED=$$(jq "with_entries(.value |= sort_by(.Namespace, .ResourceName))" "$$file"); \
		CURRENT_FILE=$$(jq . "$$file"); \
		if [ "$$CURRENT_FILE" != "$$SORTED" ]; then \
			if [ "$$PRINT_ERR" = 1 ]; then \
				echo "The following JSON files are not sorted:"; \
				PRINT_ERR=0; \
			fi; \
			echo "\t$$file"; \
		fi; \
	done; \
	if [ "$$PRINT_ERR" = 0 ]; then \
		echo "Run the following command to sort all files recursively: make sort-exception-files"; \
	fi; \