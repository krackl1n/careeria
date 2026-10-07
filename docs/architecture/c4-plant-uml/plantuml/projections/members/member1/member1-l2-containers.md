# Контейнеры области

Область Member 1: кандидаты и вакансии. Основные контейнеры: Candidate Service, Candidate DB, Vacancy Service и Vacancy DB.

Общий вход в систему: `Пользователь → Web Application → BFF → API Gateway`.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/projections/members/member1/member1-l2-containers.puml
@enduml
```

[Контекст области](member1-l1-context.md) · [Командная интеграция](../../../integration/team.md)
