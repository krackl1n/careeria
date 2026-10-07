# Контейнеры области

Область Member 2: компании и найм. Основные контейнеры: Company Service, Company DB, Hiring Service и Hiring DB.

Общий вход в систему: `Пользователь → Web Application → BFF → API Gateway`.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/projections/members/member2/member2-l2-containers.puml
@enduml
```

[Контекст области](member2-l1-context.md) · [Командная интеграция](../../../integration/team.md)
