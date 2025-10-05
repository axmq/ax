.PHONY: test unit_test test_race


unit_test:
	go test ./... -v

test:
	go test ./... -covermode=atomic

test_race:
	go test ./... --race

fmt:
	@echo "Formatting code..."
	@go tool gofumpt -l -w .