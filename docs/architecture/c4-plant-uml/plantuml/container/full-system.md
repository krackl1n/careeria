# Полная система

Представление L2 раскрывает внутреннее устройство Careeria: точки входа, доменные сервисы, их данные и платформенную инфраструктуру.

## Путь пользовательского запроса

`Browser → Web Application → BFF → API Gateway → доменный сервис → PostgreSQL`.

Каждый сервис владеет своими данными. Синхронные обращения идут через API владельца; событийные взаимодействия используют Kafka.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/container/full-system.puml
@enduml
```

## Детальные представления

- [Коммуникации](../projections/communication.md).
- [Событийная архитектура](../projections/event-driven.md).
