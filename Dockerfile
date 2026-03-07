FROM docker.io/golang:1.26.1-trixie@sha256:f6751d823c26342f9506c03797d2527668d095b0a15f1862cddb4d927a7a4ced AS base


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

FROM cgr.dev/chainguard/static:latest

COPY --from=builder /workspace/forgejo-pkg-cleanup /opt/

ENTRYPOINT [ "/opt/forgejo-pkg-cleanup" ]
