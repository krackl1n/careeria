# Careeria

Careeria — рекрутинговая платформа полного цикла: профиль кандидата, вакансии, отклики, этапы найма, оценивание, коммуникации, видеособеседования, файлы и уведомления.

## Целевой стек

- Frontend: Next.js, React, TypeScript, Tailwind CSS, TanStack Query, React Hook Form и Zod.
- Backend: Go, gRPC, Protobuf, `grpc-gateway`, `pgx`, `sqlc` и Goose.
- Данные: PostgreSQL, transactional outbox и MinIO/S3 API.
- События: Apache Kafka, Avro, Apicurio Registry и Debezium PostgreSQL Connector.
- Identity и доступ: OAuth 2.0/OIDC, JWT, Keycloak и OpenFGA.
- Realtime и медиа: Centrifugo/WebSocket и LiveKit/WebRTC.
- Наблюдаемость: OpenTelemetry, Prometheus, Grafana и Loki.

Подробное описание контейнеров, протоколов, владения данными и командной сегментации находится в [документации C4-PlantUML](docs/architecture/c4-plant-uml/README.md).

## Документация

```sh
make docs
```

Команда собирает Docker-образ с MkDocs, PlantUML и Java, запускает готовый сайт на `http://localhost:18882` и открывает браузер. Изменения Markdown и PlantUML автоматически обновляют сайт. Остановка — `make docs-stop`.

Для документации нужен Docker с Compose. Node.js, Python и Java на рабочей машине не требуются.

```sh
make docs-build              # Статический сайт в .site-docs/
make docs-test               # Проверка страниц, схем и ссылок
make architecture-validate   # Проверка исходников PlantUML
cp docs/.env.example docs/.env  # Один раз: локальные настройки сайта
make docs
```

- [Обзор документации](docs/index.md).
- [Как писать страницы и подключать диаграммы](docs/contributing.md).
- Конфигурация сайта: `mkdocs.yml`; зависимости: `docs/requirements.txt`.
- Локальные настройки контейнера: `docs/.env` (шаблон — `docs/.env.example`).
- Схемы C4-PlantUML: `docs/architecture/c4-plant-uml/plantuml/`.
- Дополнительная модель Structurizr: `docs/architecture/c4/`; запуск — `make structurizr-architecture`, проверка — `make structurizr-architecture-validate`.
