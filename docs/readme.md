Рекрутинговая система
1. Идея проекта
Рекрутинговая платформа для взаимодействия кандидатов и работодателей.
Система должна покрывать полный цикл найма:
* создание профиля кандидата;
* создание и публикация вакансий;
* отклики на вакансии;
* управление этапами найма;
* общение кандидатов и рекрутеров;
* проведение онлайн-собеседований;
* видеоконференции;
* оценка кандидатов;
* тестовые задания;
* Live Coding;
* хранение файлов;
* уведомления;
* отслеживание полной истории процесса найма.
Ключевая особенность — интеграция инструментов найма и проведения собеседований в одной системе.

2. Бизнес-сервисы
Candidate Service
Отвечает за данные кандидата.
Основные сущности:
* Candidate Profile
* Resume
* Work Experience
* Education
* Skills
* Languages
* Candidate Preferences
Candidate Service не отвечает за отклики, вакансии или процесс найма.

Company Service
Отвечает за работодателей и принадлежность пользователей к компаниям.
Основные сущности:
* Company
* Company Member
* Recruiter
* Company Role
Пример ролей:
* OWNER
* ADMIN
* RECRUITER
* HIRING_MANAGER
Рекрутер является обычным пользователем системы, связанным с определённой компанией.

Vacancy Service
Отвечает за вакансии.
Основные сущности:
* Vacancy
* Requirements
* Skills
* Employment Conditions
* Compensation
* Publication Status
Пример жизненного цикла вакансии:
DRAFT
  ↓
PUBLISHED
  ↓
PAUSED
  ↓
CLOSED
  ↓
ARCHIVED
Vacancy Service не хранит отклики и состояние кандидатов.

Hiring Service
Центральный сервис процесса найма.
Основные сущности:
* Application
* Hiring Pipeline
* Hiring Stage
* Stage Transition
* Interview
* Offer
* Rejection
* Withdrawal
* Hiring History
Главный агрегат:
Application
- id
- candidate_id
- vacancy_id
- company_id
- current_stage
- status
- created_at
- updated_at
Hiring Service является владельцем state machine процесса найма.
Пример:
APPLIED
   ↓
SCREENING
   ↓
INTERVIEW
   ↓
ASSESSMENT
   ↓
OFFER
   ↓
HIRED
Терминальные состояния:
REJECTED
WITHDRAWN
CANCELLED
HIRED
Только Hiring Service может менять состояние Application.
Каждый переход сохраняется в истории:
ApplicationTransition
- id
- application_id
- from_stage
- to_stage
- reason
- changed_by
- changed_at
- metadata
Это позволяет восстановить полный timeline процесса найма и строить аналитику по длительности этапов и воронке найма.

Assessment Service
Отвечает за оценку кандидатов.
Основные сущности:
* Assessment
* Assessment Template
* Assessment Session
* Test
* Question
* Practical Task
* Live Coding Session
* Submission
* Score
* Evaluation
Assessment Service оценивает кандидата, но не изменяет состояние Application напрямую.
Например:
Assessment Service
      │
      │ AssessmentCompleted
      ▼
     Kafka
      │
      ▼
Hiring Service
      │
      ▼
State Machine

Communication Service
Отвечает за коммуникацию пользователей.
Основные сущности:
* Conversation
* Participant
* Message
* Attachment
* Reaction
* Read Receipt
Поддерживает:
* Candidate ↔ Recruiter
* групповые чаты;
* чат во время интервью;
* realtime messages.
Realtime-доставка осуществляется через Centrifugo.

3. Платформенные сервисы
Identity Service
Отвечает за идентичность пользователя и authentication.
Функции:
* регистрация;
* авторизация;
* credentials;
* OAuth/OIDC;
* access tokens;
* refresh tokens;
* sessions;
* восстановление доступа.
Identity не должен хранить бизнес-данные кандидата или компании.

OpenFGA / ReBAC
Используется для relationship-based authorization.
Пример отношений:
user:123 recruiter company:42

vacancy:501 parent company:42

application:900 parent vacancy:501

interview:1001 parent application:900
Это позволяет выполнять проверки:
can user:123 edit vacancy:501?

can user:123 view application:900?

can user:123 manage interview:1001?
OpenFGA является источником истины для authorization relationships.

AuthZ Projection Service
Read-optimized проекция authorization-данных.
Используется для быстрых проверок прав без необходимости выполнять сетевой запрос к OpenFGA на каждый пользовательский request.
OpenFGA
   │
   ▼
AuthZ Projection
   │
   ▼
fast permission lookup
Projection является производным состоянием и может быть полностью перестроена.

Notification Service
Отвечает за доставку уведомлений.
Поддерживаемые каналы:
* in-app;
* email;
* push;
* при необходимости SMS.
Пример:
InterviewScheduled
       │
       ▼
Notification Service
       │
       ├── email
       ├── push
       └── in-app

File Service
Централизованная работа с файлами.
Функции:
* upload;
* download;
* metadata;
* presigned URL;
* validation;
* virus scanning;
* lifecycle;
* интеграция с S3 / MinIO.
Используется для:
* резюме;
* аватаров;
* логотипов компаний;
* вложений в чатах;
* assessment-файлов;
* portfolio;
* записей интервью.
Доменный сервис хранит только file_id.
Например:
Resume
- id
- candidate_id
- file_id
Физические файлы хранятся в S3 / MinIO.

4. Edge Layer
BFF
Backend for Frontend.
Используется между frontend и backend-сервисами.
Ответственность:
* работа с пользовательской сессией;
* работа с access/refresh token;
* добавление authentication context;
* агрегация данных нескольких сервисов;
* frontend-specific API;
* адаптация backend DTO под UI.
BFF не содержит бизнес-логику Hiring, Vacancy или Assessment.

API Gateway
Общая точка входа во внутренний backend.
Ответственность:
* routing;
* rate limiting;
* request limits;
* tracing;
* logging;
* observability;
* инфраструктурные middleware.

5. Realtime и видеоконференции
Centrifugo
Используется как WebSocket/realtime transport.
Примеры realtime-событий:
message.created
message.read
application.stage.changed
interview.started
assessment.updated
typing.started
Типовой поток:
Backend Service
      │
      ▼
Centrifugo
      │
      ▼
Frontend
Centrifugo не является источником бизнес-данных.

LiveKit
Используется для видеоконференций.
Поддерживает:
* WebRTC;
* audio;
* video;
* screen sharing;
* rooms;
* participants;
* recording.
Hiring Service хранит бизнес-сущность Interview.
LiveKit хранит техническую media-session.
Hiring Service
     │
     │ create room / issue token
     ▼
LiveKit API
Media traffic идёт напрямую:
Frontend
   │
   │ WebRTC
   ▼
LiveKit
и не проходит через BFF или доменные сервисы.

6. Event-Driven Architecture
Kafka
Kafka используется как основная event bus для асинхронного взаимодействия сервисов.
Примеры domain events:
UserCreated
CompanyMemberAdded
VacancyPublished
ApplicationCreated
ApplicationStageChanged
InterviewScheduled
InterviewCancelled
AssessmentAssigned
AssessmentCompleted
CandidateHired
FileReady
Kafka используется именно для событий между bounded context'ами.
Она не заменяет обычные синхронные API.

7. CDC
Debezium
Debezium используется для Change Data Capture.
Типовой поток:
PostgreSQL
   │
   │ WAL
   ▼
Debezium
   │
   ▼
Kafka
   │
   ▼
Consumers / Projections
CDC может использоваться для:
* построения read-model;
* синхронизации проекций;
* аналитики;
* индексации;
* технической интеграции между системами.
Бизнес-события при этом желательно публиковать явно.
Например:
ApplicationStageChanged
лучше является полноценным domain event, а не вычисляется на стороне consumer из изменений таблицы.

8. Контракты между сервисами
Protocol Buffers
.proto используется как основной формат контрактов для синхронного service-to-service взаимодействия.
Пример:
Service A
   │
   │ gRPC
   ▼
Service B
Контракт определяется в:
*.proto

gRPC
Используется для синхронного взаимодействия внутренних backend-сервисов там, где требуется RPC.

OpenAPI
Используется для HTTP REST API.

Swagger
Используется для визуального представления и тестирования OpenAPI-контрактов.
Swagger UI должен генерироваться автоматически.

9. Kafka-схемы
Avro
События Kafka сериализуются в Avro.
Например:
ApplicationStageChanged
имеет формальную Avro Schema.

Schema Registry
Используется для:
* хранения Avro-схем;
* версионирования;
* проверки compatibility;
* управления эволюцией событий.
Поток:
Producer
   │
   ▼
Avro Schema
   │
   ▼
Schema Registry
   │
   ▼
Kafka
   │
   ▼
Consumer

10. Документация
Документация должна преимущественно генерироваться автоматически из контрактов.
Используются:
* OpenAPI;
* Swagger UI;
* Proto documentation;
* Avro schemas;
* Schema Registry;
* документация Kafka events;
* архитектурная документация.
Это уменьшает вероятность рассинхронизации между реализацией и документацией.

11. Хранилища
Основные инфраструктурные хранилища:
PostgreSQL
S3 / MinIO
Каждый бизнес-сервис должен владеть собственными данными.
Например:
Candidate DB
Company DB
Vacancy DB
Hiring DB
Assessment DB
Communication DB
При этом один физический PostgreSQL cluster допустим, но ownership данных остаётся разделённым.

12. CI/CD
Используемый стек:
Git
Jenkins
Docker
Kubernetes
Argo CD
Процесс:
Git
 │
 ▼
Jenkins
 │
 ├── Build
 ├── Unit Tests
 ├── Integration Tests
 ├── Lint
 ├── Contract Validation
 ├── Proto Validation
 ├── Avro Compatibility Check
 │
 ▼
Docker Image
 │
 ▼
Container Registry
 │
 ▼
Deployment configuration
 │
 ▼
Argo CD
 │
 ▼
Kubernetes
Jenkins отвечает за CI.
Argo CD отвечает за GitOps CD.
Kubernetes используется как runtime-платформа.

13. Полный список бизнес-сервисов
Candidate Service
Company Service
Vacancy Service
Hiring Service
Assessment Service
Communication Service
14. Полный список платформенных сервисов
Identity Service
OpenFGA / ReBAC
AuthZ Projection Service
Notification Service
File Service
15. Edge
BFF
API Gateway
16. Realtime / Media
Centrifugo
LiveKit
17. Event Infrastructure
Kafka
Debezium CDC
Schema Registry
Avro
18. API / Contracts
Protocol Buffers
gRPC
OpenAPI
Swagger
19. Storage
PostgreSQL
S3 / MinIO
20. CI/CD
Git
Jenkins
Docker
Kubernetes
Argo CD
21. Ключевые архитектурные границы
Candidate
    → данные кандидата

Company
    → работодатель и рекрутеры

Vacancy
    → описание и lifecycle вакансии

Hiring
    → процесс найма и state machine

Assessment
    → оценка кандидата

Communication
    → пользовательская коммуникация

Identity
    → authentication

OpenFGA
    → authorization

Notification
    → доставка уведомлений

File
    → работа с файлами
Таким образом каждый сервис имеет собственную зону ответственности и владельца данных, а межсервисная интеграция строится через синхронные gRPC-контракты и асинхронные Kafka-события.
