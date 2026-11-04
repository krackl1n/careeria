# Верхнеуровневый обзор (L2)

Обзор L2 выделяет точки входа и основные сервисы без детализации хранилищ. Пользовательский запрос проходит через Web Application, BFF и API Gateway к нужному доменному сервису.

```plantuml
@startuml
!include architecture/c4/plantuml/high-level(L2)/high-level.puml
@enduml
```

Хранилища и дополнительные связи раскрыты в [полной системе](../system-context%28L1%29/full-system%28L2%29/index.md). Асинхронные потоки вынесены в [event-driven](../event-driven%28L2%29/index.md).
