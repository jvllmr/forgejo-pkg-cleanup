FROM docker.io/golang:1.27.1-trixie@sha256:433790e515d27dc6003e847e644cc0af956985cf315c1c58a3b73ee2dd305183 AS base


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

FROM cgr.dev/chainguard/static:latest@sha256:f51c2493951313c3ad4069080b2814ffb6ed6fe3909dabeb84a9482f42d5600b

COPY --from=builder /workspace/forgejo-pkg-cleanup /opt/

LABEL org.opencontainers.image.vendor="Jan Vollmer <jan@vllmr.dev>"
LABEL org.opencontainers.image.authors="Jan Vollmer <jan@vllmr.dev>"
ENTRYPOINT [ "/opt/forgejo-pkg-cleanup" ]
