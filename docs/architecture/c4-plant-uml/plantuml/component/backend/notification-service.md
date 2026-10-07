# Notification Service — компоненты L3

Notification Service принимает синхронные запросы через gRPC, читает триггерные события из Kafka и создаёт фоновые задачи доставки. Он хранит шаблоны, задачи и in-app Inbox в собственной PostgreSQL-базе; отправку email, push и SMS изолирует адаптер провайдеров SendGrid, FCM и Twilio.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/component/backend/notification-service.puml
@enduml
```

[Компонентные диаграммы](../README.md) · [Company Service](company-service.md)
