# Careeria — архитектура C4 в PlantUML

Этот каталог содержит архитектурную модель рекрутинговой платформы Careeria в нотации C4-PlantUML. Модель описывает системный контекст L1, контейнеры L2, доменные представления, динамические сценарии и общую интеграционную схему.

Технологии ниже являются целевым стеком проекта. Они фиксируют архитектурные решения для реализации и защиты лабораторных работ.

## Назначение системы

Careeria объединяет полный процесс найма:

- ведение профиля и резюме кандидата;
- создание и публикацию вакансий;
- отклики и управление этапами найма;
- тестирование, практические задания и Live Coding;
- переписку кандидата с работодателем;
- видеособеседования;
- файлы и вложения;
- уведомления;
- интеграцию с корпоративными HR-системами.

Пользователи системы: кандидат, рекрутер, нанимающий руководитель и администратор компании.

## Технологический стек

| Слой | Технологии | Назначение |
|---|---|---|
| Web Application | Next.js, React, TypeScript, Tailwind CSS | App Router, SSR/CSR, пользовательский интерфейс и адаптивная вёрстка |
| Работа с API во frontend | TanStack Query, React Hook Form, Zod | server state, формы и runtime-валидация DTO |
| Realtime и интервью во frontend | WebSocket client, LiveKit React SDK | сообщения, статусы, видеосвязь и демонстрация экрана |
| BFF | Go, `net/http`, `grpc-go`, OAuth 2.0/OIDC | HttpOnly-сессии, CSRF-защита, token refresh и агрегация API |
| API Gateway | Go, `grpc-gateway`, `grpc-go`, OpenTelemetry | REST-to-gRPC, rate limiting, request ID и distributed tracing |
| Микросервисы | Go, gRPC, Protobuf | Типизированные синхронные контракты между контейнерами |
| Доступ к PostgreSQL | `pgx`, `sqlc`, Goose | Пулы соединений, типобезопасные запросы и миграции |
| Транзакционные данные | PostgreSQL | Отдельная схема/БД на сервис, ACID и transactional outbox |
| Событийная платформа | Apache Kafka в режиме KRaft | Доменные события, consumer groups и повторная обработка |
| Схемы событий | Avro, Apicurio Registry | Версионирование и backward compatibility событий |
| CDC | Debezium PostgreSQL Connector | Чтение outbox через logical replication и публикация в Kafka |
| Авторизация | OpenFGA с PostgreSQL | ReBAC-проверки для компаний, вакансий, откликов и интервью |
| Файлы | MinIO, S3 API, MinIO Go SDK | Multipart upload, presigned URL и lifecycle объектов |
| Realtime | Centrifugo, WebSocket, JWT | Каналы, presence и доставка пользовательских событий |
| Медиа | LiveKit, WebRTC | Аудио, видео, screen sharing и запись интервью |
| Наблюдаемость | OpenTelemetry, Prometheus, Grafana, Loki | Трейсы, метрики, дашборды и централизованные логи |

Все прикладные backend-контейнеры реализуются на Go. Исключения — готовые инфраструктурные продукты: PostgreSQL, Kafka, Debezium, Apicurio Registry, OpenFGA, MinIO, Centrifugo и LiveKit.

## Внешние системы

- **Identity Provider:** Keycloak по OAuth 2.0/OpenID Connect. Identity Service отвечает за интеграцию, локальные сессии и проверку токенов.
- **Notification Providers:** SendGrid для email, Firebase Cloud Messaging для push и Twilio для SMS.
- **Employer HR System:** корпоративная HR-система, подключаемая через подписанные HTTPS webhooks и REST API.

## Доменные области и файлы команды

Технические имена файлов сохраняют привязку к участникам команды. Отображаемые заголовки диаграмм используют названия доменных областей.

### Member 1 — домен кандидатов и управления вакансиями

- `containers/member1/member1-l1-context.puml`
- `containers/member1/member1-l2-containers.puml`
- контейнеры: Candidate Service, Candidate DB, Vacancy Service и Vacancy DB;
- пользователи: кандидат и рекрутер;
- ответственность: профили, резюме, карьерные предпочтения, создание, публикация и поиск вакансий.

### Member 2 — домен компаний и процесса найма

- `containers/member2/member2-l1-context.puml`
- `containers/member2/member2-l2-containers.puml`
- контейнеры: Company Service, Company DB, Hiring Service и Hiring DB;
- пользователи: кандидат, рекрутер, нанимающий руководитель и администратор компании;
- ответственность: компании, корпоративные роли, отклики, state machine найма, интервью, предложения и история переходов.

### Member 3 — домен оценки, коммуникаций, realtime-взаимодействия и медиа

- `containers/member3/member3-l1-context.puml`
- `containers/member3/member3-l2-containers.puml`
- контейнеры: Assessment Service/DB, Communication Service/DB, Realtime Publisher, Centrifugo и LiveKit;
- пользователи: кандидат, рекрутер и нанимающий руководитель;
- ответственность: тесты, задания, Live Coding, сообщения, WebSocket-доставка и видеособеседования.

### Member 4 — платформенный домен идентификации, авторизации и инфраструктурных сервисов

- `containers/member4/member4-l1-context.puml`
- `containers/member4/member4-l2-containers.puml`
- контейнеры: Identity Service/DB, AuthZ Synchronization Service, OpenFGA, Notification Service, File Service, MinIO, Integration Service, Kafka, Apicurio Registry и Debezium;
- пользователи: все четыре роли Careeria;
- ответственность: authentication, authorization, уведомления, файлы, события и внешние интеграции.

В каждом L2 повторяется общий путь `Пользователь → Web Application → BFF → API Gateway`. Для областей Member 1–3 также показывается общая зависимость `API Gateway → Identity Service → Identity Provider`.

## Контейнеры и конкретные обязанности

- **Candidate Service:** Go/gRPC-сервис профилей, резюме, опыта, образования, навыков и предпочтений кандидата.
- **Vacancy Service:** Go/gRPC-сервис требований, условий, компенсации и жизненного цикла вакансии.
- **Company Service:** Go/gRPC-сервис компаний, участников, рекрутеров и корпоративных ролей.
- **Hiring Service:** владелец `Application` и state machine найма; только этот сервис меняет этап отклика.
- **Assessment Service:** тесты, практические задания, Live Coding, submissions и evaluations; состояние найма напрямую не изменяет.
- **Communication Service:** диалоги, сообщения, реакции, вложения и read receipts.
- **Identity Service:** OAuth 2.0/OIDC, JWT, access/refresh tokens, Argon2id credentials и пользовательские сессии.
- **AuthZ Synchronization Service:** Go-консьюмер Kafka, преобразующий доменные события в OpenFGA tuples.
- **Notification Service:** Go-консьюмер Kafka и адаптеры SendGrid, FCM и Twilio.
- **File Service:** multipart upload, presigned URL, проверка MIME/размера, antivirus hook и MinIO SDK.
- **Integration Service:** валидация подписи webhook, нормализация payload, публикация Kafka events и retry доставки.
- **Realtime Publisher:** Kafka consumer, преобразующий доменные события в пользовательские каналы Centrifugo.

## Синхронное и асинхронное взаимодействие

Синхронные пользовательские запросы проходят по цепочке:

```text
Browser → Next.js → Go BFF → Go API Gateway → Go gRPC service → PostgreSQL
```

Контракты внутренних API описываются в Protobuf. Внешний HTTP API преобразуется в gRPC через `grpc-gateway`.

Доменные события публикуются через transactional outbox:

```text
Go service → PostgreSQL transaction + outbox
           → Debezium logical replication
           → Kafka topic (Avro)
           → consumer service
```

Сервис не публикует событие напрямую после коммита: бизнес-изменение и запись outbox выполняются в одной транзакции PostgreSQL. Это исключает потерю события между сохранением данных и отправкой в Kafka.

## Данные и границы владения

- каждый бизнес-сервис владеет собственной PostgreSQL-схемой или отдельной БД;
- прямой доступ одного сервиса к таблицам другого запрещён;
- синхронное чтение выполняется через gRPC API владельца;
- асинхронная синхронизация выполняется через Kafka events;
- Avro-схемы событий регистрируются в Apicurio Registry;
- физические файлы хранятся в MinIO, доменный сервис хранит только `file_id`;
- OpenFGA хранит отношения доступа, но не бизнес-сущности.

## Безопасность

- браузер работает с HttpOnly/Secure/SameSite cookie BFF, а не хранит refresh token в JavaScript;
- BFF выполняет CSRF-защиту и ротацию сессии;
- API Gateway валидирует доверенный identity context и применяет rate limiting;
- Identity Service интегрируется с Keycloak и выпускает/проверяет JWT в рамках выбранного token flow;
- доменные сервисы выполняют resource-level проверки через OpenFGA;
- presigned URL MinIO имеют ограниченное время жизни и область доступа;
- входящие webhook проверяются по подписи и защищаются от повторной доставки через idempotency key.

## Архитектурные представления

- `integration.puml` — интегрированные L1 и L2 для всей команды;
- `views/system-context.puml` — общий системный контекст;
- `views/high-level.puml` — обзор без БД и событийной инфраструктуры;
- `views/hiring-domain.puml` — компании, вакансии, найм и оценивание;
- `views/candidate-domain.puml` — кандидаты, вакансии, identity и файлы;
- `views/communication.puml` — коммуникации, realtime, файлы и уведомления;
- `views/event-driven.puml` — Kafka, Debezium, Avro и transactional outbox;
- `views/big-picture.puml` — все контейнеры без наиболее шумных технических связей;
- `views/previous.puml` — полная схема со всеми контейнерами и связями;
- `views/dynamic-*.puml` — обработка webhook и доставка доменного события.

## Проверка и генерация

```sh
make plantuml-architecture-validate
make plantuml-architecture
```

SVG создаются в `out/`. Персональные диаграммы называются `member1-l1.svg` … `member4-l2.svg`. Каталог `out/` исключён из Git.
