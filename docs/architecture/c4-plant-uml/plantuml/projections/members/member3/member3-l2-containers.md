# Контейнеры области

Область Member 3: оценка и коммуникации. Основные контейнеры: Assessment Service/DB, Communication Service/DB, Realtime Publisher, Centrifugo и LiveKit.

Общий вход в систему: `Пользователь → Web Application → BFF → API Gateway`.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/projections/members/member3/member3-l2-containers.puml
@enduml
```

[Контекст области](member3-l1-context.md) · [Командная интеграция](../../../integration/team.md)
