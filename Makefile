SHELL := /bin/bash
DOCKER_IMAGE := 574097476646.dkr.ecr.eu-central-1.amazonaws.com/tools/markdown-link-check:7

.PHONY: go-build
go-build:
	@go build -o cmd/markdown-link-check cmd/main.go

.PHONY: go-test
go-test:
	@go test -race -cover -covermode=atomic -timeout=1m ${ARGS} ./...

.PHONY: go-lint
go-lint:
	@golangci-lint run -c misc/golangci/config.toml ./...

.PHONY: docker-build
docker-build:
	@docker build -t $(DOCKER_IMAGE) -f misc/docker/Dockerfile .

.PHONY: docker-push
docker-push:
	@docker push $(DOCKER_IMAGE)
