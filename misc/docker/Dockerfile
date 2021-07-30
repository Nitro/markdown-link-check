FROM golangci/golangci-lint:v1.41 AS golangci
FROM goreleaser/goreleaser:v0.174.1 AS goreleaser
FROM golang:1.16-buster

SHELL ["/bin/bash", "-c"]

RUN apt-get update \
  && apt-get install -y --no-install-recommends chromium=90.* \
  && rm -rf /var/lib/apt/lists/* \
  && wget --quiet https://github.com/hadolint/hadolint/releases/download/v2.6.0/hadolint-Linux-x86_64 -O /usr/local/bin/hadolint \
  && chmod +x /usr/local/bin/hadolint \
  && mkdir /tmp/tparse \
  && wget --quiet https://github.com/mfridman/tparse/releases/download/v0.8.3/tparse_0.8.3_Linux_x86_64.tar.gz -O /tmp/tparse/tparse.tar.gz \
  && tar -xvf /tmp/tparse/tparse.tar.gz -C /tmp/tparse \
  && mv /tmp/tparse/tparse /usr/local/bin/tparse \
  && chmod +x /usr/local/bin/tparse \
  && rm -Rf /tmp/tparse

COPY --from=golangci /usr/bin/golangci-lint /go/bin/
COPY --from=goreleaser /usr/local/bin/goreleaser /go/bin/
