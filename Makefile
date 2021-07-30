SHELL := /bin/bash
DOCKER_IMAGE := gonitro/markdown-link-check:8

.PHONY: go-build
go-build:
	@go build -o cmd/markdown-link-check cmd/main.go

.PHONY: go-test
go-test:
	@go test -race -cover -covermode=atomic -timeout=5m ${ARGS} ./...

.PHONY: go-lint
go-lint:
	@golangci-lint run -c misc/golangci/config.toml ./...

.PHONY: docker-build
docker-build:
	@docker build -t $(DOCKER_IMAGE) -f misc/docker/Dockerfile .

.PHONY: docker-push
docker-push:
	@docker push $(DOCKER_IMAGE)
