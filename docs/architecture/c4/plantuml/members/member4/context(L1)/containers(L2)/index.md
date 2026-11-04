# Контейнеры области (L2)

Область Member 4: платформенные сервисы. Основные контейнеры: Identity Service, AuthZ Synchronization Service, OpenFGA, File Service, MinIO, Notification Service, Integration Service и событийная инфраструктура.

Integration Service принимает подписанные вебхуки через raw HTTP route API Gateway и доставляет исходящие HTTPS webhooks внешним системам. Integration DB хранит подключения, inbox, subscriptions, deliveries и attempts. Входящее нормализованное событие фиксируется в outbox вместе с inbox и публикуется через Debezium; прямой небезопасный publish после SQL commit не используется. Kafka Events Consumer планирует исходящие доставки, Schema Registry предоставляет ему схемы. Управление подключениями компании проверяется через OpenFGA; подпись стороннего отправителя проверяется отдельно от пользовательской аутентификации.

Общий вход в систему: `Пользователь → Web Application → BFF → API Gateway`.

```plantuml
@startuml
!include architecture/c4/plantuml/members/member4/context(L1)/containers(L2)/member4-l2-containers.puml
@enduml
```

[Контекст области](../index.md) · [Integration Service · L3](../../../../system-context%28L1%29/full-system%28L2%29/integration-service%28L3%29/index.md) · [Приём HR-webhook](../../../../dynamic/hr-integration.md) · [Полная система](../../../../system-context%28L1%29/full-system%28L2%29/index.md)
