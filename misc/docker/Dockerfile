FROM golangci/golangci-lint:v1.26.0 AS golangci
FROM goreleaser/goreleaser:v0.133 AS goreleaser
FROM golang:1.14.2-buster

SHELL ["/bin/bash", "-c"]

RUN wget https://github.com/hadolint/hadolint/releases/download/v1.17.6/hadolint-Linux-x86_64 -O /usr/local/bin/hadolint \
  && chmod +x /usr/local/bin/hadolint

COPY --from=golangci /usr/bin/golangci-lint /go/bin/
COPY --from=goreleaser /bin/goreleaser /go/bin/
