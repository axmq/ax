.PHONY: test unit_test test_race integration_test test_all

test_all: unit_test test_race integration_test

unit_test:
	go test ./... -v

test:
	go test -covermode=atomic -v $(or $(path),./...)

test_race:
	go test ./... --race

integration_test:
	go test -covermode=atomic -tags=integration ./... -v

fmt:
	@echo "Formatting code..."
	@go tool gofumpt -l -w .