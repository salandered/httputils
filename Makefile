## audit
.PHONY: audit
audit:
	go mod tidy -diff
	go mod verify
	golangci-lint run ./...
	go test ./...
