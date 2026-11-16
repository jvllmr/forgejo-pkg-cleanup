FROM docker.io/golang:1.27.4-trixie@sha256:b54b2639de8c1bfe538ec5ffc45f29b1932235ac5f6d63c059853646677112d9 AS base

WORKDIR /workspace

FROM base AS builder
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY main.go .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build go build

FROM cgr.dev/chainguard/static:latest@sha256:9e2306b457ff71ce370043dc68c7b52bb1e6622c1c6c6714edbed57496f3116e

COPY --from=builder --chmod=0755 /workspace/forgejo-pkg-cleanup /opt/

LABEL org.opencontainers.image.vendor="Jan Vollmer <jan@vllmr.dev>"
LABEL org.opencontainers.image.authors="Jan Vollmer <jan@vllmr.dev>"
ENTRYPOINT [ "/opt/forgejo-pkg-cleanup" ]
