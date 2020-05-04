build:
	@go build -o cmd/markdown-link-check cmd/main.go
test:
	@gotest ./... -cover -v github.com/rakyll/hey
lint:
	@golint -set_exit_status ./...
