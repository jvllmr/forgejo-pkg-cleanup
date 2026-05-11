FROM docker.io/golang:1.26.3-trixie@sha256:d08bf3ed2bd263088ca8e23fefaf10f1b71769f6932f0a4017ba28d2a5baf001 AS base


WORKDIR /workspace

FROM base AS builder
WORKDIR /workspace
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY main.go .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build go build
RUN chmod +x /workspace/forgejo-pkg-cleanup

FROM cgr.dev/chainguard/static:latest@sha256:d6a97eb401cbc7c6d48be76ad81d7899b94303580859d396b52b67bc84ea7345

COPY --from=builder /workspace/forgejo-pkg-cleanup /opt/

LABEL org.opencontainers.image.vendor="Jan Vollmer <jan@vllmr.dev>"
LABEL org.opencontainers.image.authors="Jan Vollmer <jan@vllmr.dev>"
ENTRYPOINT [ "/opt/forgejo-pkg-cleanup" ]
