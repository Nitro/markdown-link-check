build:
	@go build -o cmd/markdown-link-check cmd/main.go
test:
	@go test ./... -cover -v
lint:
	@golint -set_exit_status ./...
