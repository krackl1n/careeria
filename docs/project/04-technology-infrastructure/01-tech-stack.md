# 4.1. Технологический стек

Все прикладные backend-сервисы Careeria реализуются на **Go**, а клиентское веб-приложение — на **Next.js / React / TypeScript**.

## Сводная таблица технологий по слоям

| Слой | Технологии | Назначение |
|---|---|---|
| **Web Application** | Next.js (App Router), React, TypeScript, Tailwind CSS | Маршрутизация, SSR/CSR и адаптивный пользовательский интерфейс |
| **Состояние и формы во Frontend** | TanStack Query, React Hook Form, Zod | Управление server state, формами и runtime-валидация DTO |
| **Realtime и медиа во Frontend** | WebSocket client, LiveKit React SDK | Мгновенные сообщения, статусы, видеосвязь и демонстрация экрана |
| **BFF** | Go, `net/http`, `grpc-go`, OAuth 2.0 / OIDC | HttpOnly-сессии, CSRF-защита, обновление токенов и агрегация ответов для UI |
| **API Gateway** | Go, `grpc-gateway`, `grpc-go`, OpenTelemetry | Преобразование HTTP/JSON в gRPC, rate limiting, request ID и трассировка |
| **Микросервисы** | Go, gRPC, Protocol Buffers | Строго типизированное синхронное межсервисное взаимодействие |
| **Работа с БД** | PostgreSQL, `pgx`, `sqlc`, Goose | Отдельная схема/БД на сервис, типобезопасные SQL-запросы и миграции |
| **Событийная платформа** | Apache Kafka (KRaft), Avro, Apicurio Registry | Асинхронная шина доменных событий, реестр схем и контроль совместимости |
| **CDC и Outbox** | Transactional Outbox, Debezium PostgreSQL Connector | Чтение WAL через logical replication и гарантированная публикация в Kafka |
| **Аутентификация и авторизация** | Keycloak (OIDC), OpenFGA (ReBAC) | Единый вход, JWT-токены и проверка графа отношений доступа |
| **Хранение файлов** | MinIO / S3 API, MinIO Go SDK | Multipart upload, presigned URL и управление жизненным циклом файлов |
| **Realtime и видеосвязь** | Centrifugo (WebSocket), LiveKit (WebRTC) | Пользовательские realtime-каналы и медиа-комнаты собеседований |
| **Наблюдаемость** | OpenTelemetry, Prometheus, Grafana, Loki | Распределённые трейсы, метрики, дашборды и централизованный сбор логов |

