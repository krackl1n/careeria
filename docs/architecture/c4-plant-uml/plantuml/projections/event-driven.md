# Событийное взаимодействие

Срез L2 показывает передачу доменных событий и подписчиков Kafka.

## Как проходит событие

1. Доменный сервис записывает бизнес-изменение и outbox в транзакции PostgreSQL.
2. Debezium читает outbox через WAL и публикует событие в Kafka; Avro-схемы хранятся в Apicurio Registry.
3. Подписчики обрабатывают события: найм, уведомления, realtime, синхронизация отношений доступа и внешние интеграции.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/projections/event-driven.puml
@enduml
```

Пошаговые примеры: [доставка уведомления](../dynamic/domain-event.md) и [обработка HR-webhook](../dynamic/hr-integration.md).
