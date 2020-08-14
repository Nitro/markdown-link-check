FROM golangci/golangci-lint:v1.28.3 AS golangci
FROM goreleaser/goreleaser:v0.140 AS goreleaser
FROM golang:1.15.0-buster

SHELL ["/bin/bash", "-c"]

RUN apt-get update \
  && apt-get install -y --no-install-recommends chromium=83.* \
  && rm -rf /var/lib/apt/lists/*

RUN wget https://github.com/hadolint/hadolint/releases/download/v1.17.6/hadolint-Linux-x86_64 -O /usr/local/bin/hadolint \
  && chmod +x /usr/local/bin/hadolint

RUN mkdir /tmp/tparse \
  && wget https://github.com/mfridman/tparse/releases/download/v0.7.4/tparse_0.7.4_Linux_x86_64.tar.gz -O /tmp/tparse/tparse.tar.gz \
  && tar -xvf /tmp/tparse/tparse.tar.gz -C /tmp/tparse \
  && mv /tmp/tparse/tparse /usr/local/bin/tparse \
  && chmod +x /usr/local/bin/tparse \
  && rm -Rf /tmp/tparse

COPY --from=golangci /usr/bin/golangci-lint /go/bin/
COPY --from=goreleaser /bin/goreleaser /go/bin/
