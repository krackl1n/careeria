# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26.5

FROM golang:${GO_VERSION}-bookworm AS build

WORKDIR /workspace

ENV CGO_ENABLED=0

# This service uses local Go modules through replace directives.  The build
# context must therefore be the repository root.
COPY packages/go/config ./packages/go/config
COPY packages/go/clock ./packages/go/clock
COPY packages/go/closer ./packages/go/closer
COPY packages/go/contracts ./packages/go/contracts
COPY packages/go/ctxmeta ./packages/go/ctxmeta
COPY packages/go/grpc ./packages/go/grpc
COPY packages/go/idgen ./packages/go/idgen
COPY packages/go/jwt ./packages/go/jwt
COPY packages/go/kafka ./packages/go/kafka
COPY packages/go/logger ./packages/go/logger
COPY packages/go/metrics ./packages/go/metrics
COPY packages/go/migration ./packages/go/migration
COPY packages/go/outbox ./packages/go/outbox
COPY packages/go/postgres ./packages/go/postgres
COPY packages/go/runtime ./packages/go/runtime
COPY packages/go/shutdown ./packages/go/shutdown
COPY packages/go/tracing ./packages/go/tracing
COPY packages/go/txmanager ./packages/go/txmanager
COPY services/identity ./services/identity

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    mkdir -p /out && \
    go build -C services/identity -trimpath -ldflags="-s -w" -o /out/identity ./cmd/service && \
    go build -C services/identity -trimpath -ldflags="-s -w" -o /out/identity-migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

COPY --from=build /out/identity /app/identity
COPY --from=build /out/identity-migrate /app/identity-migrate
COPY --from=build /workspace/services/identity/configs /app/configs

ENV IDENTITY_CONFIG_FILE=/app/configs/config.yaml

USER nonroot:nonroot

FROM runtime AS service
ENTRYPOINT ["/app/identity"]

FROM runtime AS migrate
ENTRYPOINT ["/app/identity-migrate"]
