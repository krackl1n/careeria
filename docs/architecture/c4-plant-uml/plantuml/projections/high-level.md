# Верхнеуровневый обзор

Обзор L2 выделяет точки входа и основные сервисы без детализации хранилищ. Пользовательский запрос проходит через Web Application, BFF и API Gateway к нужному доменному сервису.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/projections/high-level.puml
@enduml
```

Хранилища и дополнительные связи раскрыты в [полной системе](../container/full-system.md). Асинхронные потоки вынесены в [event-driven](event-driven.md).
