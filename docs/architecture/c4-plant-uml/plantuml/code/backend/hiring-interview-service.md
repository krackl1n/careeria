# Interview Service (Hiring Domain) — детализация кода L4

Этот раздел описывает внутреннее устройство, классы, доменные сущности, контракты и структуры компонента **Interview Service**, входящего в контейнер `Hiring Service`.

> Планирование и проведение собеседований, фиксация оценок и управление видео-комнатами.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/code/backend/hiring-interview-service.puml
@enduml
```

## 1. Архитектурные границы и принципы

1. **Изоляция Bounded Context:** Компонент владеет исключительно своими агрегатами и доменной логикой.
2. **Чистая архитектура (Ports & Adapters):** Вся внешняя инфраструктура (база данных, другие сервисы, брокер) представлена исключительно через исходящие интерфейсы.
3. **Транзакционная атомарность:** Оркестрация сценариев мутации агрегатов выполняется в рамках транзакций через Unit of Work (`TransactionManager`).

## 2. Доменная модель (Domain Layer)

### Корень агрегата `Interview` (Aggregate Root)

| Поле | Тип | Назначение |
|---|---|---|
| `ID` | `UUID` | Доменное свойство агрегата |
| `ApplicationID` | `UUID` | Доменное свойство агрегата |
| `InterviewType` | `InterviewType` | Доменное свойство агрегата |
| `Format` | `MeetingFormat` | Доменное свойство агрегата |
| `Status` | `InterviewStatus` | Доменное свойство агрегата |
| `ScheduledStartTime` | `Timestamp` | Доменное свойство агрегата |
| `ScheduledEndTime` | `Timestamp` | Доменное свойство агрегата |
| `MeetingRoomURL` | `string` | Доменное свойство агрегата |
| `Participants` | `List<Participant>` | Доменное свойство агрегата |
| `Feedbacks` | `List<Feedback>` | Доменное свойство агрегата |
| `Version` | `int` | Доменное свойство агрегата |
| `CreatedAt` | `Timestamp` | Доменное свойство агрегата |
| `UpdatedAt` | `Timestamp` | Доменное свойство агрегата |

**Ключевые методы агрегата:**

* `+SetMeetingRoomURL(url: string, now: Timestamp): void`
* `+Complete(now: Timestamp): error`
* `+Cancel(reason: string, now: Timestamp): error`
* `+SubmitFeedback(feedback: Feedback, now: Timestamp): error`

### Дочерние сущности (Entities)

#### Сущность `Feedback`

* `InterviewerUserID: UUID`
* `Score: int`
* `Notes: string`
* `SubmittedAt: Timestamp`

#### Сущность `Participant`

* `UserID: UUID`
* `Role: ParticipantRole`

## 3. Сценарии использования (Application Layer)

Прикладной сервис `HiringInterviewService` оркестрирует выполнение бизнес-сценариев:

* `+CancelInterview(ctx: Context, input: CancelInterviewInput): error`
* `+CompleteInterview(ctx: Context, input: CompleteInterviewInput): error`
* `+GetInterview(ctx: Context, interviewID: UUID, actorID: UUID): Interview`
* `+GetMeetingAccess(ctx: Context, input: GetMeetingAccessInput): MeetingAccessOutput`
* `+ListInterviewsByApplication(ctx: Context, appID: UUID, actorID: UUID): List<Interview>`
* `+ScheduleInterview(ctx: Context, input: ScheduleInterviewInput): UUID`
* `+SubmitFeedback(ctx: Context, input: SubmitFeedbackInput): error`

### Стратегии прикладного уровня (RoomStrategy)

Для оркестрации провижининга комнат используется объект стратегии `RoomStrategy` с реестром реализаций `map[MeetingFormat]MeetingRoomStrategy`.

Интерфейс стратегии `MeetingRoomStrategy` реализуют:

* `OnsiteStrategy`
* `LiveKitStrategy`

## 4. Внешние порты и контракты (Outbound Ports)

### Порт `ApplicationStageChecker`

* `+GetStage(ctx: Context, appID: UUID): PipelineStage`

### Порт `AuthorizationPort`

* `+CanManageInterview(ctx: Context, actorID: UUID, appID: UUID): error`
* `+CanSubmitFeedback(ctx: Context, actorID: UUID, interviewID: UUID): error`

### Порт `ConferenceClient`

* `+CreateRoom(ctx: Context, roomName: string): string`
* `+EndRoom(ctx: Context, roomName: string): error`

### Порт `InterviewRepository`

* `+Create(ctx: Context, it: Interview): error`
* `+Update(ctx: Context, it: Interview): error`
* `+GetByID(ctx: Context, id: UUID): Interview`
* `+ListByApplicationID(ctx: Context, appID: UUID): List<Interview>`

### Порт `OutboxRepository`

* `+Save(ctx: Context, message: Message): error`

