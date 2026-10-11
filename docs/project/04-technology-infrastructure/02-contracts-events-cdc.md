# 4.2. Контракты API, события и CDC

## Синхронные контракты (Protobuf, gRPC, OpenAPI)

Синхронные запросы проходят по цепочке:

```text
Browser ──▶ Next.js ──▶ Go BFF ──▶ Go API Gateway ──▶ Go gRPC Service ──▶ PostgreSQL
```

- **Protocol Buffers (`.proto`)** — единый источник контрактов для синхронного service-to-service взаимодействия.
- **gRPC** — транспорт для вызовов между внутренними сервисами.
- **OpenAPI / Swagger UI** — автоматически генерируемая из `.proto` спецификация внешнего REST API для визуального просмотра и тестирования.

---

## Событийная архитектура (Kafka, Avro, Schema Registry)

Асинхронное взаимодействие между bounded contexts строится через **Apache Kafka**.

**Основные доменные события:** `UserCreated`, `CompanyMemberAdded`, `VacancyPublished`, `ApplicationCreated`, `ApplicationStageChanged`, `InterviewScheduled`, `InterviewCancelled`, `AssessmentAssigned`, `AssessmentCompleted`, `CandidateHired`, `FileReady`.

- События сериализуются в формат **Avro**.
- **Apicurio Schema Registry** хранит версии Avro-схем и проверяет backward compatibility перед публикацией.

---

## Transactional Outbox и Debezium CDC

Чтобы исключить потерю событий между коммитом в БД и отправкой в брокер, бизнес-изменение и запись в таблицу `outbox` выполняются в одной транзакции PostgreSQL:

```text
Go Service ──▶ PostgreSQL (Business Data + Outbox в одной транзакции)
           ──▶ Debezium PostgreSQL Connector (WAL logical replication)
           ──▶ Apache Kafka (Avro)
           ──▶ Consumer Services / Projections
```

См. также:

- [Событийное взаимодействие (проекция C4 L2)](../../architecture/c4-plant-uml/plantuml/projections/event-driven.md)
- [Сценарий доставки доменного события](../../architecture/c4-plant-uml/plantuml/dynamic/domain-event.md)

