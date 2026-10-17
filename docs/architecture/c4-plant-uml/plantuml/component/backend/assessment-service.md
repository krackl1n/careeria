# Assessment Service — компоненты L3

Assessment Service владеет каталогом шаблонов испытаний (квизы, тестовые задания, Live Coding), сессиями прохождения тестирования кандидатами, отправленными решениями (submissions), результатами автоматических проверок и экспертными рецензиями. Медиапотоки аудио/видео взаимодействия и совместного редактирования в реальном времени обслуживаются внутренним медиасервером LiveKit (WebRTC), файлы тестов и артефакты решений хранятся в S3-совместимом Object Storage через File Service, а вся бизнес-логика, конечный автомат сессий и результаты оценивания — в Assessment DB (PostgreSQL).

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/component/backend/assessment-service.puml
@enduml
```

## Границы и статус проекта

Это целевая спецификация и каталог сценариев для проектирования L3-компонентов. Сервис изолирует функционал оценки компетенций соискателей от общей воронки найма, позволяя независимо развивать алгоритмы проверки заданий, интеграцию с раннерами кода и интерактивные интервью.

В одном backend-контейнере выделены два бизнес-компонента:

- **Assessment Template Service** — управление библиотекой заданий, тестами, языковыми средами, лимитами времени, критериями проверки и стратегиями скоринга.
- **Session Execution Service** — управление конечным автоматом индивидуальных сессий кандидатов (таймеры, доступ, дедлайны), прием решений, запуск автопроверки, координация Live Coding сессий (комнаты LiveKit) и фиксация экспертных ревью.

Оба компонента функционируют в рамках единого контейнера на Go, обращаются к единой Assessment DB и предоставляют объединенный контракт AssessmentService. Инварианты вычислений и алгоритмы оценки отделены от сущностей хранения с применением паттерна Strategy (Стратегия).

| Данные / действие | Владелец | Граница |
|---|---|---|
| Банк заданий, тесты, сессии кандидатов, решения, протоколы выполнения, баллы | Assessment Service | Не хранит резюме соискателей и общие профили компаний. |
| Назначение испытания на этапе отбора и перевод кандидата в воронке | Hiring Service | Hiring Service связывает отклик с сессией; Assessment Service фиксирует и отдает итоговый балл. |
| Потоковое аудио, видео, screen sharing и data-каналы Live Coding | LiveKit | Assessment Service создает комнаты и выпускает токены; медиапотоки через сервис не идут. |
| Изолированная компиляция, сборка и исполнение кода (Sandbox) | Code Execution Runner | Assessment Service отправляет исходный код на проверку и получает статус/метрики. |
| Файлы тестов, большие входные датасеты, zip-архивы решений | File Service / Object Storage | Assessment Service хранит только file ID тестовых пакетов и решений. |
| Пользователи, их системные роли и учетные записи | Identity Service | Идентификация кандидатов и нанимающих менеджеров из identity context. |
| Разрешения на оценку кандидатов конкретной вакансии | OpenFGA | Проверка отношений ревьюера и нанимающей команды через ReBAC. |
| Оповещение кандидата о назначении задания и дедлайне | Notification Service | Асинхронная доставка email/push по событию AssessmentAssigned. |

Kafka, Apicurio Schema Registry и Debezium обеспечивают асинхронный экспорт доменных событий через паттерн Transactional Outbox.

## Элементы диаграммы

### Внутренние компоненты

| Элемент | Технология / слой | Ответственность | Зависимости |
|---|---|---|---|
| gRPC Server | Go / gRPC | Валидация входных структур запросов, распаковка identity context, маппинг ошибок FSM в gRPC-статусы. | Assessment Template Service, Session Execution Service |
| Assessment Template Service | Go Application Service | Управляет шаблонами испытаний, задачами, тест-кейсами и привязкой стратегий расчета баллов. | Authorization Client, Template Repo, Transactional Outbox |
| Session Execution Service | Go Application Service | Управляет жизненным циклом сессии, контролирует дедлайны через Session Lifecycle Policy, запускает расчет баллов. | Session Lifecycle Policy, Scoring Context, Sandbox Client, LiveKit Manager, Session Repo, Authorization Client, Transactional Outbox |
| Scoring Context & Strategies | Go Domain Model / Strategy | Реализует семейство алгоритмов вычисления итогового балла (Weighted, Strict Auto, Live Coding Rubric). | Отсутствуют (чистый Go / Domain) |
| Session Lifecycle Policy | Go Domain Policy | Межагрегатная валидация: проверяет не истекло ли время сдачи, разрешен ли запуск, допустим ли переход состояний. | Отсутствуют (чистый Go) |
| Authorization Client | Go / OpenFGA SDK | Проверяет права рекрутеров на создание шаблонов и интервьюеров на внесение оценок. | OpenFGA |
| Template Repo | Go / pgx, sqlc | Чтение и запись шаблонов, задач и связанных правил проверки в PostgreSQL. | Assessment DB |
| Session Repo | Go / pgx, sqlc | Чтение и запись сессий, попыток сдачи кода, результатов тестов и ручных рецензий. | Assessment DB |
| Sandbox Client | Go / gRPC | Взаимодействует с изолированным раннером безопасного исполнения кода кандидатов. | Code Execution Runner |
| LiveKit Manager | Go / LiveKit Go SDK | Инициализирует WebRTC-комнаты в LiveKit, генерирует ограниченные токены для кандидата и экзаменатора. | LiveKit |
| Transactional Outbox | Go / pgx, sqlc | Сохраняет доменные события (AssessmentCompleted и др.) в рамках бизнес-транзакций. | Assessment DB |
| Session Expiry Worker | Go Background Worker | Фоновый процесс, находящий просроченные сессии и переводящий их в состояние EXPIRED. | Session Repo, Transactional Outbox |

### Соседние контейнеры

| Элемент | Технология | Назначение / связь |
|---|---|---|
| API Gateway | Go, grpc-gateway, grpc-go | Проксирует клиентские запросы и передает метаданные пользователя. |
| Assessment DB | PostgreSQL 16 | Хранение шаблонов, сессий, решений, рецензий, outbox и ключей идемпотентности. |
| OpenFGA | OpenFGA, PostgreSQL | Авторизация доступа нанимающих инженеров к сессиям конкретной вакансии. |
| LiveKit | LiveKit, WebRTC, JWT | Обеспечивает интерактивную видеосвязь и синхронизацию состояния редактора в Live Coding. |
| Code Execution Runner | gRPC, Docker/gVisor | Изолированная «песочница» для сборки и прогона юнит-тестов на решениях кандидатов. |
| File Service | Go, gRPC | Загрузка и выдача ссылок на архивы тестов и репозиториев решений. |

## Доменная модель и хранение

Модель декомпозирована на изолированные агрегаты с вынесением логики расчетов в стратегии:

| Сущность | Основные данные | Инварианты |
|---|---|---|
| AssessmentTemplate | template ID, company ID, title, type (QUIZ / PRACTICAL_CODE / LIVE_CODING), scoringStrategy, timeLimitMinutes, passingThreshold, version, timestamps | Шаблон содержит минимум одну задачу. Лимит времени: от 10 минут до 72 часов. Пороговый балл не может превышать 100%. |
| AssessmentTask | task ID, template ID, title, instructions, allowedLanguages, testSuiteRef, maxScore | Языки из поддерживаемого системного allowlist. Ссылка на тест-сьют указывает на валидный архив в File Service. |
| AssessmentSession | session ID, template ID, candidate ID, applicationId, status, validUntil, startedAt, expiresAt, submittedAt, finalScore, isPassed, version | Переходы состояний строго по конечному автомату (FSM). Поле expiresAt рассчитывается в момент старта сессии: startedAt + timeLimitMinutes. Попытка сдачи после expiresAt отклоняется. |
| TaskSubmission | submission ID, session ID, task ID, language, sourceCode, status (PENDING / RUNNING / PASSED / FAILED / COMPILATION_ERROR), metrics, createdAt | Решение принимается только в рамках активной сессии (IN_PROGRESS). Метрики включают реальное время работы (мс) и RAM (МБ). |
| ManualReview | review ID, session ID, reviewer ID, criteriaScores (JSONB), feedback, createdAt | Ревью может оставить только назначенный интервьюер. Баллы по критериям нормализованы в диапазоне 0..100. |
| OutboxEvent | event ID, aggregate ID, aggregate type, event type, payload, occurredAt | Гарантирует надежную публикацию событий изменения статусов испытаний в Kafka. |

### Паттерн «Стратегия» (Strategy) для расчета оценки

В строгом соответствии с требованиями ЛР3 кафедры, расчет итогового балла динамически подменяется через паттерн «Стратегия»:

- **WeightedScoringStrategy:** вычисляет интегральный балл по формуле с весами: например, 60% — автотесты раннера, 40% — ручная оценка критериев кода (архитектура, чистота, алгоритм).
- **StrictAutoScoringStrategy:** 100% оценка по результатам автотестов; сессия считается пройденной только при прохождении всех обязательных тестов (Screening-задачи).
- **LiveCodingRubricStrategy:** оценка на основе матрицы компетенций (Problem Solving, Communication, Code Quality, Edge Cases), заполняемой нанимающим инженером в реальном времени.

Предлагаемые таблицы: assessment_templates, assessment_tasks, assessment_sessions, task_submissions, assessment_manual_reviews, idempotency_keys, assessment_outbox.

Межсервисные связи опираются на внешние UUID (candidate_id, application_id, company_id) без cross-database foreign keys.

## Статусы и жизненный цикл

| Ось | Предлагаемые значения | Правило |
|---|---|---|
| Жизненный цикл сессии | SCHEDULED, IN_PROGRESS, SUBMITTED, EVALUATED, EXPIRED | SCHEDULED — назначено, ждет старта. IN_PROGRESS — кандидат выполняет, таймер запущен. SUBMITTED — решения сданы, идет проверка. EVALUATED — финальный балл рассчитан. EXPIRED — время вышло до сдачи. |
| Статус решения (Submission) | PENDING, RUNNING, PASSED, FAILED, COMPILATION_ERROR, TIMEOUT | Фиксирует текущий статус выполнения кода в Sandbox Runner. |
| Итоговый вердикт | PENDING, PASSED, FAILED | Вычисляется доменной стратегией скоринга относительно passing_threshold. |

### Диаграмма состояний сессии кандидата

```text
[SCHEDULED] ──(StartSession)──► [IN_PROGRESS] ──(FinalizeSession)──► [SUBMITTED] ──(CalculateScore)──► [EVALUATED]
     │                                │
     └───(validUntil timeout)─────────┴──(expiresAt timeout)──────► [EXPIRED]
```

## Каталог сценариев

| № | Сценарий | Компонент | Результат |
|---|---|---|---|
| 1 | CreateTemplate | Assessment Template Service | Создан шаблон испытания с привязкой задач и стратегии. |
| 2 | UpdateTemplate | Assessment Template Service | Обновлены формулировки задач и тесты без изменения активных сессий. |
| 3 | AssignAssessment | Session Execution Service | Кандидату назначено испытание в статусе SCHEDULED. |
| 4 | GetMyAssignedAssessments | Session Execution Service | Список назначенных тестов для авторизованного кандидата. |
| 5 | StartSession | Session Execution Service | Сессия переведена в IN_PROGRESS, зафиксирован дедлайн expires_at. |
| 6 | GetSessionDetails | Session Execution Service | Получены задачи и условия выполнения активной сессии. |
| 7 | SubmitTaskSolution | Session Execution Service | Код решения передан на асинхронную проверку в Sandbox Runner. |
| 8 | PollSubmissionResult | Session Execution Service | Получен статус прохождения юнит-тестов и метрики потребления ресурсов. |
| 9 | InitializeLiveCoding | Session Execution Service | Создана комната LiveKit, возвращены JWT-токены подключения. |
| 10 | FinalizeSession | Session Execution Service | Кандидат завершил испытание; переход в статус SUBMITTED. |
| 11 | SubmitManualReview | Session Execution Service | Экзаменатор внес оценки по критериям и обратную связь. |
| 12 | CalculateSessionScore | Session Execution Service | Стратегия скоринга вычисляет итоговый результат; статус EVALUATED. |
| 13 | GetAssessmentReport | Session Execution Service | Итоговый детальный отчет для нанимающего менеджера. |
| 14 | ReconcileExpiredSessions | Session Expiry Worker | Автоматический перевод зависших или просроченных сессий в EXPIRED. |

### 3. AssignAssessment — назначить испытание кандидату

**Инициатор и вход:** рекрутер или система найма (Hiring Service), template_id, candidate_id, application_id, valid_until, idempotency key.

**Проверки и выполнение:** проверяются права рекрутера через OpenFGA, валидность шаблона и дата доступности (valid_until в будущем). В БД создается AssessmentSession в статусе SCHEDULED. Записывается событие AssessmentAssigned в outbox.

**Результат:** возвращается session_id.

### 5. StartSession — начать прохождение испытания

**Инициатор и вход:** кандидат, session_id.

**Проверки:** сессия принадлежит кандидату (candidate_id == identity.user_id), текущий статус строго SCHEDULED, текущее время меньше valid_until.

**Выполнение:** SessionExecutionService фиксирует started_at = NOW(), вычисляет expires_at = NOW() + template.time_limit_minutes, переводит статус в IN_PROGRESS и инкрементирует version. Событие AssessmentStarted сохраняется в outbox. Возвращаются дедлайн и тексты заданий (без скрытых тест-кейсов).

### 7. SubmitTaskSolution — отправить код на проверку

**Инициатор и вход:** кандидат, session_id, task_id, language, source_code.

**Проверки:** сессия находится в статусе IN_PROGRESS, дедлайн expires_at не наступил (SessionLifecyclePolicy), язык входит в список разрешенных.

**Выполнение:** создается запись task_submissions со статусом PENDING. Сервис отправляет асинхронную команду в Sandbox Client. Раннер запускает контейнер в изоляции, выполняет сборку и прогоняет тесты. При получении ответа запись обновляется статусом (PASSED или FAILED), метриками (время, память) и детализацией упавших assert-тестов.

### 9. InitializeLiveCoding — инициализировать сессию интервью

**Инициатор и вход:** нанимающий инженер или кандидат, session_id.

**Проверки:** шаблон относится к типу LIVE_CODING, участник является кандидатом либо авторизованным интервьюером.

**Выполнение:** LiveKit Manager обращается к API LiveKit, создавая защищенную WebRTC-комнату careeria-assessment-{session_id}. Генерируется персонализированный JWT с правами публикации аудио/видео и доступа к data-каналам синхронизации кода. Токен и URL возвращаются клиенту.

### 10–12. FinalizeSession и CalculateSessionScore — фиксация и скоринг

**Инициатор и вход:** кандидат (завершение) либо ревьюер (оценка), session_id.

**Выполнение:** при переходе в SUBMITTED кандидат больше не может отправлять код. После получения необходимых ручных рецензий (или сразу при 100% автопроверке) вызывается ScoringContext. В соответствии с настройкой шаблона загружается нужная реализация ScoringStrategy.

Стратегия вычисляет итоговый процент успешности и булев флаг is_passed. Сессия обновляется полями final_score, is_passed, статус переходит в EVALUATED. В outbox пишется событие AssessmentCompleted.

## Транзакции, безопасность и интеграция

### Авторизация внутри бизнес-сервисов

Для соискателя доступ к прохождению сессии валидируется по прямому совпадению session.candidate_id == identity.user_id. Для сотрудников компаний (просмотр решений, выставление оценок, создание шаблонов) проверяется ReBAC через OpenFGA: кортеж user:<id>#interviewer@application:<id>. Кандидату никогда не возвращаются скрытые тесты и эталонные решения.

### SQL и внешние операции

Сохранение решений и перевод статусов изолированы от внешних раннеров кода. Сетевой вызов к Sandbox Client или LiveKit выполняется вне транзакции БД. Результат возвращается отдельной короткой транзакцией фиксации статуса решения.

### Предлагаемые события

| События | Изменение | Минимальный payload |
|---|---|---|
| AssessmentAssigned | Назначено испытание | session ID, candidate ID, application ID, template ID, valid_until |
| AssessmentStarted | Кандидат начал выполнение | session ID, candidate ID, started_at, expires_at |
| AssessmentSubmitted | Решения отправлены | session ID, candidate ID, submitted_at |
| AssessmentCompleted | Итоговый расчет выполнен | session ID, application ID, candidate ID, final_score, is_passed, strategy_used |
| AssessmentExpired | Истек срок выполнения | session ID, candidate ID, expired_at |

События через transactional outbox и Debezium передаются в топик Kafka careeria.assessment.events.v1 в формате Avro.

Ключевой потребитель — **Hiring Service**: при получении события AssessmentCompleted он проверяет результат (is_passed) и автоматически переводит отклик кандидата по воронке найма на этап формирования оффера или отсева.

### Ошибки и наблюдаемость

Специфические ошибки: SessionExpired, SessionNotInProgress, LanguageNotSupported, CompilationError, RunnerUnavailable, EvaluationAlreadyCompleted.

Метрики: длительность прохождения тестов, latency автопроверки в раннере, распределение баллов по стратегиям, частота падений по тайм-ауту. В трассировку OpenTelemetry включаются теги session.id и 
unner.execution_time.

## Что не включено автоматически

| Вопрос | Решение первой версии / следующий контракт |
|---|---|
| AI-прокторинг и анализ биометрии | Не включается. Автоматический контроль движения взгляда и детекция посторонних в кадре требуют сложного ML-контура и юридического согласия. |
| Исполнение произвольного кода в браузере (WebAssembly) | Все решения компилируются строго на бэкенде в изолированных контейнерах для предотвращения читерства. |
| Автоматическая генерация задач через LLM | Шаблоны создаются методистами и нанимающими инженерами вручную; валидация сгенерированного кода отложена на будущие релизы. |
| Нагрузочное тестирование решений соискателей | Ограничено базовыми лимитами CPU/RAM (до 2 ядер, до 512 МБ памяти на тест-ран). Задачи на распределенные системы не поддерживаются. |

[Все L3](../README.md) · [Hiring Service](hiring-service.md) · [Candidate Service](candidate-service.md) · [Notification Service](notification-service.md)
