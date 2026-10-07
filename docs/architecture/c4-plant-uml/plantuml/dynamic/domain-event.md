# Доставка доменного события

Сценарий показывает, как изменение процесса найма приводит к уведомлению пользователя.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/dynamic/domain-event.puml
@enduml
```

## Порядок действий

1. Hiring Service сохраняет данные и outbox в Hiring DB.
2. Debezium читает запись через PostgreSQL WAL.
3. Событие публикуется в Kafka в формате Avro.
4. Notification Service получает событие.
5. Внешний провайдер доставляет email, push или SMS.

Общий набор подписчиков показан в [событийной архитектуре](../projections/event-driven.md).
