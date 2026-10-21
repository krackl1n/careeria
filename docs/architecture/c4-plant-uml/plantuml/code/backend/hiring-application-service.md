# Application Service (Hiring Domain) — детализация кода L4

Этот раздел описывает внутреннее устройство, классы, доменные сущности, контракты, проекции и межагрегатные политики компонента **Application Service**, входящего в контейнер `Hiring Service`.

Диаграмма детализирует реализацию компонента на уровне кода (C4 Level 4), раскрывая автономность агрегата `JobApplication`, отдельный доменный объект проекции `InterviewSummary`, чистую межагрегатную политику `OfferEligibilityPolicy` и оркестрацию I/O в прикладном сервисе.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/code/backend/hiring-application-service.puml
@enduml
```

## Границы и архитектурные принципы компонента

1. **Строгие границы транзакционной целостности агрегата:**
   * Корень агрегата `JobApplication` владеет **только своими данными** (идентификаторы вакансии/кандидата, текущий этап, история переходов).
   * Данные собеседований не встраиваются внутрь агрегата отклика. Собеседование — это отдельный жизненный цикл.
2. **Отдельный доменный объект проекции (`InterviewSummary`):**
   * Для проверки условий продвижения кандидата используется отдельная проекция/Read-модель `InterviewSummary`.
3. **Чистая межагрегатная политика (`OfferEligibilityPolicy`):**
   * Доменная политика связывает два независимых объекта: `JobApplication` и `List<InterviewSummary>`.
   * **Политика не содержит внутри портов и не выполняет I/O.** Она является чистой функцией/структурой в памяти: `CanInitiateOffer(app, interviews)`.
4. **Прикладной сервис (Application Service) как оркестратор I/O:**
   * Сервис загружает агрегат из `ApplicationRepository`.
   * Сервис загружает проекции интервью из `InterviewSummaryRepository`.
   * Передает их в политику `OfferEligibilityPolicy`.
   * При успешной валидации отдает команду агрегату совершить переход и сохраняет результат в БД и Outbox.

---

## Доменная модель (Domain Layer)

### 1. Корень агрегата `JobApplication` (Aggregate Root)

* **Поля агрегата:**
  * `id: UUID` — уникальный идентификатор отклика.
  * `vacancyID: UUID` — ссылка на вакансию (Vacancy Service).
  * `candidateID: UUID` — ссылка на профиль кандидата (Candidate Service).
  * `currentStage: PipelineStage` — текущий этап воронки найма.
  * `version: int` — маркер оптимистической блокировки (OCC).
  * `history: List<StageTransition>` — упорядоченная неизменяемая история переходов.

* **Бизнес-методы агрегата:**
  * `MarkInterview(actorID: UUID, comment: string): error` — фиксирует переход на этап собеседований (`INTERVIEW`).
  * `MarkOfferPending(actorID: UUID): error` — переводит отклик в статус подготовки предложения (`OFFER_PENDING`).
  * `Reject(reason: RejectReason, actorID: UUID): error` — завершает процесс отказом компании с фиксацией причины.
  * `Withdraw(actorID: UUID): error` — фиксирует самостоятельный отзыв заявки кандидатом (`WITHDRAWN`).
  * `AdvanceFromAssessment(score: int): error` — переводит отклик по результатам автоматического тестирования.

### 2. Проекция `InterviewSummary` (Domain Model / Projection)

Отдельный объект предметной области, представляющий результат проведенного собеседования:
* `interviewID: UUID` — идентификатор интервью в Interview Service.
* `applicationID: UUID` — ссылка на отклик.
* `stage: string` — этап (например, "Технический скрининг", "Финальное интервью").
* `status: InterviewStatus` — статус встречи (`SCHEDULED`, `COMPLETED`, `CANCELLED`).
* `hasFeedback: bool` — флаг наличия заполненной оценки.
* `score: int` — балл (1–5).
* `completedAt: Timestamp` — время завершения.

### 3. Межагрегатная политика `OfferEligibilityPolicy` (Domain Policy)

Инкапсулирует сквозное бизнес-правило перехода кандидата на этап оффера:
```go
type OfferEligibilityPolicy struct{}

func (p OfferEligibilityPolicy) CanInitiateOffer(app *JobApplication, interviews []InterviewSummary) error {
    if app.CurrentStage() != StageInterview {
        return ErrMustBeInInterviewStage
    }

    var hasCompletedWithFeedback bool
    for _, interview := range interviews {
        if interview.Status == InterviewCompleted && interview.HasFeedback {
            hasCompletedWithFeedback = true
            break
        }
    }

    if !hasCompletedWithFeedback {
        return ErrOfferRequiresCompletedInterviewWithFeedback
    }

    return nil
}
```

### 4. Сущность `StageTransition` (Entity)

Фиксирует факт смены этапа воронки: `id: UUID`, `fromStage: PipelineStage`, `toStage: PipelineStage`, `changedBy: UUID`, `changedAt: Timestamp`, `comment: string`.

### 5. Объекты-значения (Value Objects)

* **`PipelineStage` (enum):**  
  `APPLIED` $\rightarrow$ `SCREENING` $\rightarrow$ `TECH_ASSESSMENT` $\rightarrow$ `INTERVIEW` $\rightarrow$ `OFFER_PENDING` $\rightarrow$ `OFFER_ACCEPTED` $\rightarrow$ `HIRED`. Терминальные: `REJECTED`, `WITHDRAWN`.
* **`RejectReason` (Value Object):**  
  Стандартизированный код отказа (`code`) и комментарий.

---

## Сценарии выполнения (Application Layer)

### Сценарий: `InitiateOffer` — подготовка предложения о работе

Пошаговое выполнение сценария наглядно демонстрирует разделение обязанностей:

```
[HiringApplicationService]
       │
       │ 1. Читает отклик из репозитория
       ├──> ApplicationRepository.FindByID(appID) ──> возвращает *JobApplication
       │
       │ 2. Читает проекции интервью
       ├──> InterviewSummaryRepository.FindByApplicationID(appID) ──> возвращает []InterviewSummary
       │
       │ 3. Вызывает чистую межагрегатную политику в памяти
       ├──> OfferEligibilityPolicy.CanInitiateOffer(app, interviews)
       │    └── Валидирует: этап INTERVIEW + есть COMPLETED с фидбеком
       │
       │ 4. Командует агрегату изменить статус
       ├──> app.MarkOfferPending(actorID)
       │    └── Агрегат меняет currentStage и добавляет запись в history
       │
       │ 5. Сохраняет агрегат и событие в транзакции
       └──> ApplicationRepository.Save(app) + OutboxRepository.SaveEvent(...)
```

---

## Внешние контракты (Ports & Repositories)

| Интерфейс | Назначение | Реализующий L3-компонент |
|---|---|---|
| `ApplicationRepository` | Чтение и сохранение агрегата `JobApplication` и его истории `StageTransition`. | `Application Repo` (Go / pgx, sqlc) |
| `InterviewSummaryRepository` | Выборка проекций интервью по отклику (`FindByApplicationID`). | Чтение из базы `Hiring DB` (view/таблица `interviews`). |
| `OutboxRepository` | Транзакционная запись доменных событий изменения этапа воронки в таблицу `outbox_events`. | `Transactional Outbox` (Go / pgx, sqlc) |
| `AuthorizationPort` | Проверка ReBAC-полномочий инициатора через OpenFGA. | `Authorization Client` (Go / OpenFGA SDK) |
