FROM docker.io/golang:1.26.5-trixie@sha256:4ee9ffa999b4583ce281939cdff828763083610292f252279a0cee77473bd9a7 AS base


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

FROM cgr.dev/chainguard/static:latest@sha256:fe55470f22d3259488d9d3739168d8f04da67755f0b69382bc26eda4a7d3d327

COPY --from=builder /workspace/forgejo-pkg-cleanup /opt/

LABEL org.opencontainers.image.vendor="Jan Vollmer <jan@vllmr.dev>"
LABEL org.opencontainers.image.authors="Jan Vollmer <jan@vllmr.dev>"
ENTRYPOINT [ "/opt/forgejo-pkg-cleanup" ]
