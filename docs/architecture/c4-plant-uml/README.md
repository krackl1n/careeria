# Careeria — архитектура C4 в PlantUML

Этот каталог содержит архитектурную модель рекрутинговой платформы Careeria в нотации C4-PlantUML. Модель описывает системный контекст L1, контейнеры L2, доменные представления, динамические сценарии и общую интеграционную схему.

Технологии ниже являются целевым стеком проекта. Они фиксируют архитектурные решения для реализации и защиты лабораторных работ.

## Структура каталога

```text
c4-plant-uml/
├── README.md
├── plantuml/
│   ├── common/                  # Общий визуальный язык
│   │   ├── styles.puml
│   │   ├── component-styles.puml
│   │   ├── component-wide-layout.puml
│   │   ├── tags.puml
│   │   └── legends.puml
│   ├── model/                   # Переиспользуемые элементы и связи
│   │   ├── elements.puml
│   │   ├── component-elements.puml
│   │   ├── all-elements.puml
│   │   └── all-relationships.puml
│   ├── context/                 # C4 L1
│   │   └── system-context.puml
│   ├── container/               # C4 L2: полная система
│   │   └── full-system.puml
│   ├── component/               # C4 L3: компоненты контейнеров
│   │   └── backend/company-service.puml
│   ├── projections/             # Срезы модели по теме и ответственности
│   │   ├── high-level.puml
│   │   ├── big-picture.puml
│   │   ├── event-driven.puml
│   │   ├── candidate-domain.puml
│   │   ├── hiring-domain.puml
│   │   ├── communication.puml
│   │   └── members/
│   │       ├── member1/          # member1-l1-context.puml, member1-l2-containers.puml
│   │       ├── member2/          # member2-l1-context.puml, member2-l2-containers.puml
│   │       ├── member3/          # member3-l1-context.puml, member3-l2-containers.puml
│   │       └── member4/          # member4-l1-context.puml, member4-l2-containers.puml
│   ├── integration/             # Сведение областей команды
│   │   ├── team.puml             # C4 L1 и L2 с общими include
│   │   └── standalone.puml       # Автономная версия без include
│   └── dynamic/                 # Сценарии взаимодействия
│       ├── hr-integration.puml
│       └── domain-event.puml
└── images/
    └── generated/               # Генерируемые SVG, исключены из Git
```

### Как выбирать представление

- **Уровень C4** определяет глубину описания: `context/` — системы и пользователи, `container/` — приложения, сервисы и хранилища.
- **Проекция** выбирает часть той же архитектуры: домен, область участника команды, верхнеуровневый обзор или событийное взаимодействие. Проекции member сохраняют свои L1/L2 и технические имена файлов.
- **Командная интеграция** в `integration/team.puml` объединяет области всех участников. Внешние HR-интеграции описаны в общей модели и сценарии `dynamic/hr-integration.puml`.
- **Динамический сценарий** показывает порядок взаимодействий для конкретного процесса.

Верхнеуровневая и event-driven диаграммы — тематические проекции L2. Слово «проекция» здесь обозначает архитектурное представление; сервис `AuthZ Synchronization Service` выполняет отдельную задачу обновления отношений доступа из событий.

### Расширение структуры

По мере появления соответствующих моделей добавляются:

- `context/system-landscape.puml` — ландшафт нескольких систем;
- `container/backend.puml` и `container/frontend.puml` — отдельные контейнерные представления;
- новые L3-представления в `component/` — компоненты внутри конкретного контейнера;
- `deployment/{production,local}.puml` — узлы развёртывания и размещённые экземпляры контейнеров;
- новые сценарии в `dynamic/`, названные по процессам Careeria, например `user-login.puml` и `apply-to-vacancy.puml`.

Диаграмма `component/backend/company-service.puml` уже раскрывает внутреннее устройство Company Service. Остальные L3-представления и deployment-модель будут добавляться по мере уточнения реализации. Контейнерные срезы не следует выдавать за компоненты L3, а `production` требует согласованной схемы инфраструктуры.

### Правила сопровождения

1. Переиспользуемые L1/L2-объявления элементов добавлять в `plantuml/model/elements.puml`, L3-элементы — в `plantuml/model/component-elements.puml`, общие связи — в `all-relationships.puml`.
2. L1/L2-диаграммы подключают `common/styles.puml`; L3-диаграммы подключают `common/component-styles.puml`. В нём централизованы C4 Component, скруглённый стиль, шрифт, размеры, маршрутизация `polyline` и общие теги инфраструктуры. Для широких use case L3 дополнительно подключается `common/component-wide-layout.puml`.
3. Использовать относительные `!include`. Файлы `common/` и `model/` — фрагменты без `@startuml`; остальные `.puml` — самостоятельные точки входа для генерации.
4. При переносе диаграмм сохранять идентификаторы `@startuml`: они задают имена SVG. Файл `container/full-system.puml` ранее назывался `views/previous.puml` и сохраняет имя результата `C4-L2-Previous.svg`.
5. `integration/standalone.puml` — отдельная автономная копия для обмена. При изменении модели обновлять её вместе с `integration/team.puml`; автоматически она не синхронизируется.
6. Не редактировать SVG вручную. После изменения исходников запускать проверку и генерацию из корня репозитория.

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

- `plantuml/projections/members/member1/member1-l1-context.puml`
- `plantuml/projections/members/member1/member1-l2-containers.puml`
- контейнеры: Candidate Service, Candidate DB, Vacancy Service и Vacancy DB;
- пользователи: кандидат и рекрутер;
- ответственность: профили, резюме, карьерные предпочтения, создание, публикация и поиск вакансий.

### Member 2 — домен компаний и процесса найма

- `plantuml/projections/members/member2/member2-l1-context.puml`
- `plantuml/projections/members/member2/member2-l2-containers.puml`
- контейнеры: Company Service, Company DB, Hiring Service и Hiring DB;
- пользователи: кандидат, рекрутер, нанимающий руководитель и администратор компании;
- ответственность: компании, корпоративные роли, отклики, state machine найма, интервью, предложения и история переходов.

### Member 3 — домен оценки, коммуникаций, realtime-взаимодействия и медиа

- `plantuml/projections/members/member3/member3-l1-context.puml`
- `plantuml/projections/members/member3/member3-l2-containers.puml`
- контейнеры: Assessment Service/DB, Communication Service/DB, Realtime Publisher, Centrifugo и LiveKit;
- пользователи: кандидат, рекрутер и нанимающий руководитель;
- ответственность: тесты, задания, Live Coding, сообщения, WebSocket-доставка и видеособеседования.

### Member 4 — платформенный домен идентификации, авторизации и инфраструктурных сервисов

- `plantuml/projections/members/member4/member4-l1-context.puml`
- `plantuml/projections/members/member4/member4-l2-containers.puml`
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

| Представление | Назначение |
|---|---|
| [Системный контекст](plantuml/context/system-context.puml) | Пользователи Careeria и внешние системы, C4 L1 |
| [Полная система](plantuml/container/full-system.puml) | Все контейнеры и связи, C4 L2 |
| [Верхнеуровневая проекция](plantuml/projections/high-level.puml) | Обзор без БД и событийной инфраструктуры |
| [Event-driven](plantuml/projections/event-driven.puml) | Kafka, Debezium, Avro и transactional outbox |
| [Коммуникации](plantuml/projections/communication.puml) | Коммуникации, realtime, файлы и уведомления |
| [Командная интеграция](plantuml/integration/team.puml) | Интегрированные L1 и L2 для всей команды |
| [Автономная интеграция](plantuml/integration/standalone.puml) | Два представления без зависимостей от других файлов и C4-библиотеки |
| [HR webhook](plantuml/dynamic/hr-integration.puml) | Последовательность обработки входящего webhook |
| [Доменное событие](plantuml/dynamic/domain-event.puml) | Доставка события от outbox до уведомления |

## Сайт документации

Архитектура — подраздел общей документации MkDocs с темой Material. Страницы и инструменты находятся в `docs/`, конфигурация сайта — в `mkdocs.yml` в корне репозитория. Node.js не требуется.

```sh
make docs                   # Общая документация
make docs-build             # Статическая сборка
make docs-test              # Проверка сайта
```

Для запуска нужен Docker с Compose. Образ документации содержит MkDocs, Java и PlantUML; MkDocs обновляет сайт при изменении Markdown и `.puml`.

## Описания и схемы

Markdown-файлы рядом со схемами являются самостоятельными страницами, без промежуточного генератора. Текст, таблицы, код и ссылки оформляются обычным Markdown. Блок `plantuml` в нужном месте страницы подключает схему через нативный `!include`.

Пример — [system-context.md](plantuml/context/system-context.md). Полные правила находятся в [руководстве по документации](../../contributing.md).

Для интеграционных файлов с несколькими диаграммами сохранены отдельные страницы L1/L2. Они подключают именованные секции `!startsub` / `!endsub` из исходного `.puml` через `!includesub`. Схемы не дублируются.

### Переходы внутри диаграмм

Расширение `plantuml-markdown` вставляет SVG в страницу; ссылки `$link` на элементах остаются активными. В системном контексте Careeria ведёт на `/architecture/c4-plant-uml/plantuml/container/full-system/`.

Для обычных переходов между страницами используйте относительные ссылки на `.md`. Ссылки на `.puml` открывают исходники. В навигации сайта отображаются только страницы Markdown.

## Проверка и генерация SVG

```sh
make plantuml-architecture-validate
make plantuml-architecture
```

Makefile находит все `.puml` внутри `plantuml/`, кроме фрагментов `common/` и `model/`. Новые проекции, компоненты и deployment-диаграммы автоматически попадают в обе команды. Проверка и генерация останавливаются при синтаксических ошибках.

SVG создаются в `images/generated/`. Имена определяются идентификаторами `@startuml`, например `member1-l1.svg` … `member4-l2.svg`; интеграционные файлы содержат по две диаграммы и создают по два SVG. Результаты исключены из Git, в каталоге хранится только `.gitkeep`. Старый каталог `out/`, если остался локально, больше не используется.

Structurizr сохранён как дополнительный инструмент: `make structurizr-architecture` и `make structurizr-architecture-validate`. Команда `make architecture-validate` проверяет исходники C4-PlantUML.
