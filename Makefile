SHELL := /bin/bash -eu -o pipefail

GO_TEST_FLAGS := -race -shuffle=on
COVERAGE_MIN  := 100.0


#  Tasks
#-----------------------------------------------
.PHONY: lint
lint:
	@golangci-lint run

.PHONY: fmt
fmt:
	@golangci-lint fmt

.PHONY: test
test:
	@go test $(GO_TEST_FLAGS) ./...

.PHONY: cover
cover:
	@go test $(GO_TEST_FLAGS) -covermode=atomic -coverprofile=coverage.out .
	@go tool cover -func=coverage.out
	@total="$$(go tool cover -func=coverage.out | awk '/^total:/ { print $$3 }')"; \
		if [ "$$total" != "$(COVERAGE_MIN)%" ]; then \
			echo "coverage is $$total, below the required $(COVERAGE_MIN)%" >&2; \
			exit 1; \
		fi
