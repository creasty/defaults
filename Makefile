SHELL := /bin/bash -eu -o pipefail

GO_TEST_FLAGS := -race -shuffle=on


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
	@go test $(GO_TEST_FLAGS) -covermode=atomic -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out
