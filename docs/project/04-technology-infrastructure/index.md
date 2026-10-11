# 4. Технологическая основа и инфраструктура

В главе описаны целевой технологический стек Careeria, синхронные и асинхронные контракты взаимодействия сервисов, паттерн Transactional Outbox / CDC, принципы владения данными и конвейер CI/CD.

## Содержание главы

- [4.1. Технологический стек](01-tech-stack.md) — технологии frontend, backend, данных, событий и наблюдаемости.
- [4.2. Контракты API, события и CDC](02-contracts-events-cdc.md) — Protobuf, gRPC, OpenAPI, Apache Kafka, Avro, Schema Registry и Debezium.
- [4.3. Хранилища данных и процесс CI/CD](03-storage-cicd.md) — изоляция данных в PostgreSQL, S3/MinIO и GitOps-доставка (Jenkins, Argo CD, Kubernetes).

