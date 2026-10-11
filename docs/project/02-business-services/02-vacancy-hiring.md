# 2.2. Вакансии и процесс найма

## Vacancy Service

Отвечает за публикацию и жизненный цикл вакансий. Не хранит отклики и текущие этапы кандидатов.

**Основные доменные сущности:**

- `Vacancy` — карточка и описание вакансии;
- `Requirements` и `Skills` — требования к кандидату и ключевые навыки;
- `Employment Conditions` — график, локация и формат занятости;
- `Compensation` — зарплатная вилка и валюта;
- `Publication Status` — статус жизненного цикла вакансии.

**Жизненный цикл вакансии:**

```text
DRAFT → PUBLISHED → PAUSED → CLOSED → ARCHIVED
```

Компонентная схема сервиса: [Vacancy Service — компоненты L3](../../architecture/c4-plant-uml/plantuml/component/backend/vacancy-service.md).

---

## Hiring Service

Центральный сервис процесса найма и единственный владелец state machine отклика (`Application`).

**Основные доменные сущности:**

- `Application` — главный агрегат отклика кандидата на вакансию (`id`, `candidate_id`, `vacancy_id`, `company_id`, `current_stage`, `status`, `created_at`, `updated_at`);
- `Hiring Pipeline` и `Hiring Stage` — конфигурация этапов отбора;
- `Stage Transition` (`ApplicationTransition`) — неизменяемая запись истории перехода (`id`, `application_id`, `from_stage`, `to_stage`, `reason`, `changed_by`, `changed_at`, `metadata`);
- `Interview` — бизнес-сущность собеседования;
- `Offer`, `Rejection`, `Withdrawal` — решения по итогам отбора.

**Типовая воронка этапов (State Machine):**

```text
APPLIED → SCREENING → INTERVIEW → ASSESSMENT → OFFER → HIRED
```

**Терминальные состояния:** `HIRED`, `REJECTED`, `WITHDRAWN`, `CANCELLED`.

Только Hiring Service имеет право изменять состояние `Application`. Каждый переход фиксируется в `ApplicationTransition`, что позволяет восстанавливать полный timeline найма и строить аналитику по воронке.

Компонентная схема сервиса: [Hiring Service — компоненты L3](../../architecture/c4-plant-uml/plantuml/component/backend/hiring-service.md).

