# Careeria — C4-PlantUML

PlantUML-версия архитектурных схем из `../c4`. Диаграммы используют C4-PlantUML из стандартной библиотеки PlantUML и общие макросы доменов из `model/elements.puml`.

## Представления

- `views/system-context.puml` — системный контекст.
- `views/high-level.puml` — верхнеуровневый обзор без БД и событийной инфраструктуры.
- `views/hiring-domain.puml` — найм и рекрутинг.
- `views/candidate-domain.puml` — кандидаты и вакансии.
- `views/communication.puml` — коммуникации и уведомления.
- `views/event-driven.puml` — Kafka, Debezium и transactional outbox.
- `views/big-picture.puml` — все контейнеры без сквозных OpenFGA, CDC и Schema Registry связей.
- `views/previous.puml` — полная схема со всеми контейнерами и связями.
- `views/dynamic-*.puml` — ключевые интеграционные сценарии.

## Проверка и генерация

```sh
make plantuml-architecture-validate
make plantuml-architecture
```

SVG-файлы создаются в `out/` и не добавляются в Git.
