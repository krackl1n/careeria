# Hiring Service

Сервис найма платформы **Careeria**. Отвечает за сквозной процесс найма кандидатов: обработку откликов, проведение этапов пайплайна, организацию и проведение собеседований (онлайн и офлайн), а также формирование и согласование джоб-офферов.

---

## 1. Архитектурный контекст (Bounded Contexts)

В соответствии с диаграммами C4 (`hiring-service-l3` и `hiring-application-service-l4`), внутри сервиса выделены три тесно связанных доменных агрегата, работающих в едином рантайме:

1. **Job Application (`internal/domain/application`)** — жизненный цикл отклика кандидата на вакансию, стейт-машина этапов пайплайна и аудит-лог переходов (`StageTransition`).
2. **Interview (`internal/domain/interview`)** — планирование и проведение собеседований, назначение участников с ролями (`ParticipantRole`), сбор оценок и отзывов (`Feedback`), интеграция с переговорными комнатами (`MeetingRoomStrategy`).
3. **Job Offer (`internal/domain/offer`)** — подготовка условий оффера, согласование, отправка кандидату, контроль дедлайнов экспирации и обработка решения (принят/отклонен/отозван).

```
                      ┌───────────────────────────────────────────────┐
                      │              Hiring Service                   │
                      │                                               │
                      │   ┌───────────────────────────────────────┐   │
   gRPC Client ───────┼──>│               Transport               │   │
(API Gateway / BFF)   │   └───────────────────┬───────────────────┘   │
                      │                       │                       │
                      │   ┌───────────────────▼───────────────────┐   │
                      │   │            Use Cases Layer            │   │
                      │   │   [Application] [Interview] [Offer]  │   │
                      │   └──────┬────────────────┬───────────────┘   │
                      │          │                │                   │
                      │   ┌──────▼──────┐  ┌──────▼───────────────┐   │
                      │   │ Domain Layer│  │ Strategy Registry    │   │
                      │   │  Aggregates │  │ (LiveKit vs Onsite)  │   │
                      │   │   Policies  │  └──────────────────────┘   │
                      │   │  Value Obj. │                             │
                      │   └──────┬──────┘                             │
                      │          │                                    │
                      │   ┌──────▼────────────────────────────────┐   │
                      │   │             Infrastructure            │   │
                      │   │  Postgres | Outbox | OpenFGA | LiveKit│   │
                      │   └───────────────────────────────────────┘   │
                      └───────────────────────────────────────────────┘
```

---

## 2. Ключевые инженерные правила и конвенции

### 2.1. Доменные агрегаты и сущности
* **Открытые поля без геттеров**: агрегаты объявляют экспортированные поля (`ID`, `CurrentStage`, `Status` и т.д.). Бойлерплейтные геттеры (`ID()`, `Status()`) запрещены.
* **Защита инвариантов**:
  * Создание нового объекта — **только** через валидирующий конструктор (`New`, `Schedule`, `Create`).
  * Изменение состояния — **только** через методы бизнес-логики (`ChangeStage`, `SubmitFeedback`, `Send`, `Accept` и т.д.).
  * Прямая мутация полей сущности вне доменных методов строго запрещена архитектурной конвенцией.
* **Регидрация из БД**: метод `Reconstitute` не используется; репозиторий заполняет поля структуры напрямую через литерал (`&JobApplication{...}`).

### 2.2. Запрет Primitive Obsession (Value Objects)
Сырые примитивы (`string`, `int`) для доменных понятий заменяются на строго типизированные Value Objects с инкапсулированной валидацией и поведением:
* `PipelineStage` — этапы найма (`APPLIED`, `SCREENING`, `TECH_ASSESSMENT`, `INTERVIEW`, `OFFER_PENDING`, `OFFER_ACCEPTED`, `HIRED`, `REJECTED`, `WITHDRAWN`). Содержит правила переходов `CanTransitionTo(target)` и терминальности `IsTerminal()`.
* `ParticipantRole` — роль на интервью (`LEAD_INTERVIEWER`, `INTERVIEWER`, `HIRING_MANAGER`, `RECRUITER`, `BAR_RAISER`, `OBSERVER`, `CANDIDATE`). Содержит бизнес-правило `CanEvaluate() bool` (наблюдатели и кандидаты не имеют права выставлять оценки).
* `InterviewType` — тип собеседования (`SCREENING`, `TECHNICAL`, `SYSTEM_DESIGN`, `BEHAVIORAL`, `BAR_RAISER`, `EXECUTIVE`).
* `MeetingFormat` — формат встречи (`ONLINE_LIVEKIT`, `ONSITE`, `EXTERNAL_LINK`).
* `Currency` — 3-буквенный ISO-4217 код валюты (`RUB`, `USD`, `EUR`, `CNY`...).
* `RejectCode` — типизированная причина отказа (`NOT_QUALIFIED`, `ASSESSMENT_FAILED`, `CULTURE_MISMATCH`...).

### 2.3. Дочерние Entity и Value Objects
Дочерние элементы агрегатов обязаны иметь собственные фабрики и инкапсулировать правила, чтобы юзкейсы не собирали сырые структуры:
* `StageTransition` — создается через `NewStageTransition(...)`, содержит методы `IsInitial()`, `IsTerminal()`.
* `RejectReason` — создается через `NewRejectReason(...)`, содержит `EffectiveComment()`.
* `Participant` — создается через `NewParticipant(...)`.
* `Feedback` — создается через `NewFeedback(...)`, содержит `IsPassing()`, `IsStrongPass()`.

### 2.4. Разделение ошибок по слоям
```
[Domain]                   [Use Case]                   [Transport / gRPC]
errors.New(...) ──> errors.Is(...) ──> apperr.Conflict / ──> codes.FailedPrecondition /
                                       apperr.NotFound /     errdetails.ErrorInfo
                                       apperr.InvalidArgument
```
1. **Доменный слой (`internal/domain/...`)**: оперирует исключительно стандартными чистыми sentinel errors (`errors.New`). Домен **не зависит** от пакета `apperr`.
2. **Слой юзкейсов (`internal/usecase/...`)**: проверяет доменные ошибки через `errors.Is` и преобразует их в типизированные прикладные ошибки `apperr` с кодами:
   * `apperr.Conflict` (недопустимый переход стадии, дубликат отклика, истекший оффер).
   * `apperr.NotFound` (сущность не найдена в репозитории).
   * `apperr.InvalidArgument` (некорректные параметры запроса).
   * `apperr.Forbidden` (отказ в авторизации прав ReBAC).
3. **Транспортный слой (`internal/transport/grpc`)**: перехватчик `ErrorResolver` транслирует `apperr` в стандартные статус-коды gRPC и прикрепляет Google RPC `ErrorInfo` с доменом `careeria.hiring`.

### 2.5. Комментарии в кодовой базе
* Тривиальные, очевидные и шаблонные комментарии (вроде `// 1. Schedule Interview`, `// Save to repo`) категорически запрещены.
* Код должен быть самодокументируемым за счет выразительных имен типов, функций и констант.

---

## 3. Паттерны проектирования

### 3.1. MeetingRoomStrategy & Registry (`internal/usecase/interview/room`)
Вместо жесткой привязки к конкретному сервису видеосвязи используется паттерн **Стратегия + Реестр**:

```go
type MeetingRoomStrategy interface {
    Format() interviewdomain.MeetingFormat
    Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error)
    GenerateAccess(ctx context.Context, req AccessRequest) (*AccessResult, error)
    Teardown(ctx context.Context, req TeardownRequest) error
}
```

* **`LiveKitStrategy`**: создает медиа-комнату в LiveKit, генерирует персонализированный JWT-токен на вход, при отмене закрывает комнату на сервере.
* **`OnsiteStrategy`**: валидирует адрес офиса/переговорки, генерирует для кандидата памятку и инструкции для ресепшена, teardown — no-op.
* **`Registry`**: потокобезопасный реестр, отдающий нужную стратегию по `MeetingFormat`.

### 3.2. Межагрегатная политика (`OfferEligibilityPolicy`)
Чистое доменное правило (`internal/domain/policy`), проверяющее возможность выставления оффера:
* Заявка должна находиться в стадии `INTERVIEW`.
* В переданном списке завершенных собеседований (`InterviewSummary`) должно быть минимум одно собеседование со статусом `COMPLETED` и заполненным фидбеком.
* Политика не содержит I/O; все данные загружаются юзкейсом заранее.

---

## 4. Взаимодействие сервисов (Service-to-Service)

### 4.1. Внутрипроцессные координаторы (In-Process)
Подсервисы `Application`, `Interview` и `Offer` скомпонованы в один процесс, но изолированы интерфейсными портами:
* **`ApplicationCoordinator`**: вызывается из `Offer Service` при акцепте или отклонении оффера для продвижения или отката стадии заявки (`AdvanceToOfferAccepted`, `RevertToInterview`).
* **`ApplicationStageChecker`**: вызывается из `Interview Service` для проверки допустимости планирования собеседования для текущей стадии отклика.

### 4.2. Межсервисное взаимодействие (Platform-wide)
* **Синхронные внешние вызовы**: только инфраструктурные зависимости:
  * **OpenFGA** — проверка разрешений ReBAC через gRPC SDK.
  * **LiveKit** — создание и завершение сессий видеоконференций.
* **Асинхронные бизнес-события**: прямое синхронное общение с другими бизнес-микросервисами платформы (`notification`, `candidate`, `identity`) отсутствует. Интеграция выполняется исключительно через **Transactional Outbox $\to$ Debezium CDC $\to$ Apache Kafka**:
  * `hiring.application.applied`
  * `hiring.application.stage_changed`
  * `hiring.application.rejected`
  * `hiring.application.withdrawn`
  * `hiring.interview.scheduled`
  * `hiring.interview.completed`
  * `hiring.interview.cancelled`
  * `hiring.interview.feedback_submitted`
  * `hiring.offer.created`
  * `hiring.offer.sent`
  * `hiring.offer.accepted`
  * `hiring.offer.declined`
  * `hiring.offer.revoked`
  * `hiring.offer.expired`

---

## 5. Структура директорий

```
services/hiring/
├── cmd/
│   └── service/                     # Точка входа main.go
├── configs/                         # Конфигурационные файлы YAML
├── internal/
│   ├── app/                         # Инициализация, граф зависимостей (Dependencies, App, Runtimes)
│   ├── config/                      # Загрузка и валидация конфигов
│   ├── domain/                      # Чистая бизнес-логика (без внешних зависимостей)
│   │   ├── apperr/                  # Прикладные типы ошибок (Conflict, NotFound, etc.)
│   │   ├── application/             # Агрегат JobApplication, стадии, переходы
│   │   ├── event/                   # Фабрика и интерфейсы доменных сообщений
│   │   ├── interview/               # Агрегат Interview, Feedback, ParticipantRole, InterviewType
│   │   ├── offer/                   # Агрегат JobOffer, Currency, OfferStatus
│   │   └── policy/                  # Межагрегатные политики (OfferEligibilityPolicy)
│   ├── transport/                   # gRPC сервер, хэндлеры, маппинг ошибок
│   │   └── grpc/
│   └── usecase/                     # Сценарии приложения (оркестрация I/O)
│       ├── application/             # ApplyVacancy, ChangeStage, Reject, Withdraw, PipelineHistory
│       ├── conference/              # Клиент конференций
│       ├── interview/               # ScheduleInterview, SubmitFeedback, Complete, Cancel, GetAccess
│       │   └── room/                # Стратегии комнат (MeetingRoomStrategy, LiveKit, Onsite, Registry)
│       └── offer/                   # CreateOffer, SendOffer, AcceptOffer, Decline, Revoke, Expire
└── README.md                        # Данная документация
```

---

## 6. Запуск тестов

Для прогона тестов без использования кэша:

```bash
go test -count=1 -v ./services/hiring/...
```

