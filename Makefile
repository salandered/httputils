## fmt
.PHONY: fmt
fmt:
	golangci-lint fmt ./...

## audit
.PHONY: audit
audit:
	go mod tidy -diff
	go mod verify
	golangci-lint run ./...
	go test ./...
