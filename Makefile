build:
	@go build -o cmd/markdown-link-check cmd/main.go
test:
	@go get -u github.com/rakyll/gotest
	@gotest ./... -cover -race -v github.com/rakyll/hey
lint:
	@go get -u golang.org/x/lint/golint
	@golint -set_exit_status ./...
