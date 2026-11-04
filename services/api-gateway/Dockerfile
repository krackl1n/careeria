# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26.5

FROM golang:${GO_VERSION}-bookworm AS build

WORKDIR /workspace

ENV CGO_ENABLED=0

COPY packages/go/closer ./packages/go/closer
COPY packages/go/config ./packages/go/config
COPY packages/go/contracts ./packages/go/contracts
COPY packages/go/grpc ./packages/go/grpc
COPY packages/go/http ./packages/go/http
COPY packages/go/jwt ./packages/go/jwt
COPY packages/go/ctxmeta ./packages/go/ctxmeta
COPY packages/go/logger ./packages/go/logger
COPY packages/go/metrics ./packages/go/metrics
COPY packages/go/runtime ./packages/go/runtime
COPY packages/go/shutdown ./packages/go/shutdown
COPY packages/go/tracing ./packages/go/tracing
COPY services/api-gateway ./services/api-gateway

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    mkdir -p /out && \
    go build \
        -C services/api-gateway \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/api-gateway \
        ./cmd/gateway

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/api-gateway /app/api-gateway
COPY --from=build /workspace/services/api-gateway/configs /app/configs

ENV API_GATEWAY_CONFIG_FILE=/app/configs/config.yaml

EXPOSE 8080 9090

USER nonroot:nonroot

ENTRYPOINT ["/app/api-gateway"]
