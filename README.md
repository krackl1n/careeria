# Careeria

Careeria — рекрутинговая платформа полного цикла: профиль кандидата, вакансии, отклики, этапы найма, оценивание, коммуникации, видеособеседования, файлы и уведомления.

## Целевой стек

- Frontend: Next.js, React, TypeScript, Tailwind CSS, TanStack Query, React Hook Form и Zod.
- Backend: Go, gRPC, Protobuf, `grpc-gateway`, `pgx`, `sqlc` и Goose.
- Данные: PostgreSQL, transactional outbox и MinIO/S3 API.
- События: Apache Kafka, Avro, Apicurio Registry и Debezium PostgreSQL Connector.
- Identity и доступ: OAuth 2.0/OIDC, JWT, Keycloak и OpenFGA.
- Realtime и медиа: Centrifugo/WebSocket и LiveKit/WebRTC.
- Наблюдаемость: OpenTelemetry, Prometheus, Grafana и Loki.

Подробное описание контейнеров, протоколов, владения данными и командной сегментации находится в [документации C4-PlantUML](docs/architecture/c4-plant-uml/README.md).

## Архитектурные схемы

```sh
make architecture-validate
make plantuml-architecture-validate
make plantuml-architecture
```

- Structurizr DSL: `docs/architecture/c4/`
- C4-PlantUML: `docs/architecture/c4-plant-uml/`
