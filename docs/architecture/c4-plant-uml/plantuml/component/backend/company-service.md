# Company Service — компоненты L3

Диаграмма раскрывает контейнер `Company Service` из L2. Сервис владеет компаниями, участниками компаний и корпоративными ролями; `Hiring Service` и другие сервисы получают эти данные только через его API.

Единый gRPC Server направляет команды в сценарии Company Service и Membership Service. Репозитории сохраняют агрегаты Company и Member, а transactional outbox фиксирует интеграционные события в той же транзакции PostgreSQL.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/component/backend/company-service.puml
@enduml
```

[Контейнеры компаний и найма](../../projections/members/member2/member2-l2-containers.md) · [Событийное взаимодействие](../../projections/event-driven.md)
