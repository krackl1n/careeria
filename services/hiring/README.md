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

### 3.1. MeetingRoomStrategy & RoomStrategy (`internal/usecase/interview/room`)
Вместо жесткой привязки к конкретному сервису видеосвязи используется паттерн **Стратегия (GoF Strategy + Context/Registry)**:

```go
type MeetingRoomStrategy interface {
    Format() interviewdomain.MeetingFormat
    Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error)
    GenerateAccess(ctx context.Context, req AccessRequest) (*AccessResult, error)
    Teardown(ctx context.Context, req TeardownRequest) error
}

type RoomStrategy struct {
    registry map[interviewdomain.MeetingFormat]MeetingRoomStrategy
}
```

* **`LiveKitStrategy`**: создает медиа-комнату в LiveKit через внешний порт `ConferenceClient`, генерирует JWT-токен на вход, при отмене закрывает комнату на медиа-сервере.
* **`OnsiteStrategy`**: валидирует адрес офиса/переговорки, генерирует для кандидата инструкции для ресепшена, teardown — no-op.
* **`RoomStrategy`**: контекст стратегии с внутренним реестром реализаций по `MeetingFormat`, предоставляющий сервису методы делегирования (`Provision`, `GenerateAccess`, `Teardown`, `Resolve`).

### 3.2. Межагрегатная политика (`OfferEligibilityPolicy`)
Чистое доменное правило (`internal/domain/policy`), проверяющее возможность выставления оффера:
* Заявка должна находиться в стадии `INTERVIEW`.
* В переданном списке собеседований (`Interview`) должно быть минимум одно собеседование со статусом `COMPLETED` и заполненным фидбеком.
* Политика не содержит I/O; все данные загружаются юзкейсом заранее.

---

## 4. Взаимодействие сервисов (Service-to-Service)

### 4.1. Внутрипроцессная координация и атомарность (In-Process ACID)
Подсервисы `Application`, `Interview` и `Offer` скомпонованы в один процесс сервиса `hiring`. Взаимодействие между агрегатами происходит напрямую через репозитории и доменные методы внутри единой транзакции базы данных (`txManager.Do`):
* **`Offer Service` $\leftrightarrow$ `JobApplication`**:
  * **Создание оффера (`CreateOffer`)**: если заявка находится в стадии `INTERVIEW`, агрегат переводится в `OFFER_PENDING` и сохраняется в `appRepo` атомарно с созданием черновика оффера. Больше нет необходимости вручную вызывать отдельный шаг инициации.
  * **Принятие оффера (`AcceptOffer`)**: оффер переходит в статус `ACCEPTED`, а заявка переводится в стадию `OFFER_ACCEPTED` в рамках одной транзакции БД с генерацией Outbox-события.
  * **Отклонение или отзыв (`DeclineOffer`, `RevokeOffer`)**: оффер переводится в `DECLINED` или `REVOKED`, а заявка откатывается обратно в стадию `INTERVIEW` в той же транзакции.
* **`Interview Service` $\leftrightarrow$ `JobApplication`**:
  * Вызывается порт `ApplicationStageChecker` для проверки допустимости планирования собеседования для текущей стадии отклика.

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
│   └── usecase/                     # Сценарии приложения (один сценарий = один файл)
│       ├── application/             # apply.go, change_stage.go, assessment_result.go, promote.go, initiate_offer.go, reject.go, withdraw.go, get.go, history.go
│       ├── conference/              # service.go (клиент конференций)
│       ├── interview/               # schedule.go, complete.go, cancel.go, submit_feedback.go, get.go, get_access.go
│       │   └── room/                # strategy.go, registry.go, livekit.go, onsite.go
│       └── offer/                   # create.go, send.go, accept.go, decline.go, revoke.go, expire.go, get.go
└── README.md                        # Данная документация
```

---

## 6. Запуск тестов

Для прогона тестов без использования кэша:

```bash
go test -count=1 -v ./services/hiring/...
```

