# Контейнеры области

Область Member 4: платформенные сервисы. Основные контейнеры: Identity Service, AuthZ Synchronization Service, OpenFGA, File Service, MinIO, Notification Service, Integration Service и событийная инфраструктура.

Общий вход в систему: `Пользователь → Web Application → BFF → API Gateway`.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/projections/members/member4/member4-l2-containers.puml
@enduml
```

[Контекст области](member4-l1-context.md) · [Командная интеграция](../../../integration/team.md)
