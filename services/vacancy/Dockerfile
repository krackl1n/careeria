ARG GO_VERSION=1.26.5
FROM golang:${GO_VERSION}-bookworm AS build
WORKDIR /workspace
ENV CGO_ENABLED=0

COPY packages/go/authz ./packages/go/authz
COPY packages/go/clock ./packages/go/clock
COPY packages/go/contracts ./packages/go/contracts
COPY packages/go/idgen ./packages/go/idgen
COPY packages/go/config ./packages/go/config
COPY packages/go/ctxmeta ./packages/go/ctxmeta
COPY packages/go/grpc ./packages/go/grpc
COPY packages/go/logger ./packages/go/logger
COPY packages/go/postgres ./packages/go/postgres
COPY packages/go/migration ./packages/go/migration
COPY packages/go/txmanager ./packages/go/txmanager
COPY packages/go/runtime ./packages/go/runtime
COPY packages/go/shutdown ./packages/go/shutdown
COPY services/vacancy ./services/vacancy

WORKDIR /workspace/services/vacancy
RUN go build -trimpath -o /out/ ./cmd/service ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
COPY --from=build /out/ /app/

FROM runtime AS migrate
ENTRYPOINT ["/app/migrate"]

FROM runtime AS service
EXPOSE 50054
ENTRYPOINT ["/app/service"]
