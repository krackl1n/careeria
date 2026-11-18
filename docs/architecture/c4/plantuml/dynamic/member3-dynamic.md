# Сквозной динамический процесс оценки компетенций и realtime-оповещения (Dynamic Diagram)

<!-- excerpt -->

Кандидат отправляет решение в Assessment Service. Сервис получает отчёт раннера и сохраняет оценку. Нанимающая команда получает обновление через Centrifugo.

Сначала Assessment Service проверяет право кандидата на отправку решения и передаёт код с тестами в раннер. По полученному отчёту сервис рассчитывает оценку и фиксирует её вместе с событием. Через Debezium и Kafka результат достигает подписчиков. Realtime Publisher обновляет интерфейс нанимающей команды, Hiring Service и Notification Service выполняют дальнейшую обработку.

```plantuml
@startuml
!include architecture/c4/plantuml/dynamic/member3-dynamic.puml
@enduml
```

<p align="center">Рисунок - Проверка решения и доставка результата</p>

<p align="right">Таблица - Элементы. Проверка решения и доставка результата</p>

| Элемент | Тип или технология | Назначение и ответственность |
|---|---|---|
| Кандидат | Пользователь | Проходит техническое испытание, отправляет решение в веб-интерфейсе и получает вердикт. |
| Рекрутер / Нанимающий менеджер | Пользователь | Отслеживает прогресс кандидата в реальном времени и получает WebSocket-уведомление об успешной сдаче. |
| Web Application | Next.js, React, TypeScript, Tailwind CSS | Интерфейс React работает в браузере. Серверный BFF обрабатывает пользовательскую сессию и обращается к API Gateway. Страницы и BFF развёртываются вместе в одном приложении Next.js. |
| API Gateway | Go, grpc-gateway | API Gateway принимает HTTP-запрос от BFF, проверяет предъявленный токен по открытым ключам Identity и применяет политику доступа к методу. Доменный сервис получает вызов по gRPC. |
| Assessment Service | Go, gRPC, Protobuf, pgx, sqlc | Assessment Service управляет шаблонами заданий и попытками кандидатов. Для проверки кода он вызывает Code Execution Runner, затем рассчитывает балл и сохраняет результат в Assessment DB. |
| OpenFGA | OpenFGA, PostgreSQL | Сервис проверки доступа на основе отношений между субъектом и ресурсом. Доменные сервисы запрашивают решение о доступе, а обработчик синхронизации обновляет отношения по событиям платформы. |
| Code Execution Runner | Go, gRPC, Docker / gVisor | Code Execution Runner запускает решение и тесты в изолированной среде с ограничениями времени и памяти. Отчёт получает Assessment Service, который рассчитывает итоговый балл. |
| Assessment DB | PostgreSQL 16 | Сессии, решения, оценки и transactional outbox. |
| Debezium | Debezium PostgreSQL Connector | Компонент захвата зафиксированных изменений PostgreSQL. Читает transactional outbox через журнал WAL и публикует доменные события в Kafka. |
| Kafka | Apache Kafka, KRaft | Событийная шина платформы. Хранит доменные сообщения в топиках и передаёт их группам потребителей для независимой асинхронной обработки. |
| Realtime Publisher | Go, Kafka Consumer, Centrifugo HTTP API | Realtime Publisher читает события из Kafka, определяет канал получателя и публикует сообщение в Centrifugo. Centrifugo доставляет его в браузер по WebSocket. |
| Centrifugo | Centrifugo, WebSocket, JWT | Сервис подключений браузеров и каналов обновления интерфейса. Принимает публикации Realtime Publisher и доставляет сообщения авторизованным подписчикам по WebSocket. |
| Hiring Service | Go, gRPC, Protobuf, pgx, sqlc | Hiring Service ведёт отклик от первого этапа отбора до предложения о работе. Компоненты откликов, интервью и предложений согласуют переходы между этапами и сохраняют историю в Hiring DB. |
| Notification Service | Go, Kafka Consumer | Notification Service принимает команды и события из Kafka, выбирает шаблон и создаёт задачу доставки. Обработчики сохраняют уведомления в личном кабинете или передают их провайдерам email, push и SMS. |
| Notification Providers | Система | Внешние поставщики доставки сообщений. SendGrid обрабатывает email, Firebase Cloud Messaging доставляет push, Twilio отправляет SMS по запросам сервиса уведомлений. |

<p align="right">Таблица - Связи. Проверка решения и доставка результата</p>

| Источник | Получатель | Содержание взаимодействия | Протокол или способ вызова |
|---|---|---|---|
| Кандидат | Web Application | 1-2. Отправляет решение с сессионной cookie. | HTTPS / UI |
| Web Application | API Gateway | 3. BFF передаёт авторизованный запрос. | HTTP / JSON |
| API Gateway | Assessment Service | 4. Вызывает метод SubmitTaskSolution(). Запрос и ответ передаются как сообщения Protobuf. | gRPC / HTTP2, порт 50055 |
| Assessment Service | OpenFGA | 5. Проверяет права кандидата (can_submit). Запрос и ответ передаются как сообщения Protobuf. | gRPC / mTLS, порт 8081 |
| Assessment Service | Code Execution Runner | 6. Направляет код и тесты на изолированное исполнение. Запрос и ответ передаются как сообщения Protobuf. | gRPC / HTTP2, порт 50060 |
| Code Execution Runner | Assessment Service | 7. Возвращает статус прохождения тестов и метрики CPU/RAM. Запрос и ответ передаются как сообщения Protobuf. | gRPC response |
| Assessment Service | Assessment DB | 8. Применяет ScoringStrategy, переводит FSM в EVALUATED и пишет Outbox. Получатель выполняет чтение или запись в базе данных сервиса. | SQL Transaction, порт 5432 |
| Assessment Service | API Gateway | 9. Возвращает статус OK, итоговый балл и протокол тестов. Запрос и ответ передаются как сообщения Protobuf. | gRPC response |
| API Gateway | Web Application | 10. Возвращает ответ серверному BFF. | HTTP / JSON |
| Web Application | Кандидат | 11-12. Возвращает и отображает результаты проверки. | UI Render |
| Assessment DB | Debezium | 13. Считывает запись Outbox из журнала транзакций WAL (CDC). Коннектор читает зафиксированные изменения из журнала PostgreSQL. | PostgreSQL WAL (pgoutput) |
| Debezium | Kafka | 14. Публикует AssessmentCompletedEvent в топик assessment.events. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro, порт 9092 |
| Kafka | Realtime Publisher | 15. Вычитывает событие завершения испытания. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro |
| Realtime Publisher | Centrifugo | 16. Публикует realtime-событие в канал компании (publish API). | Centrifugo HTTP API, порт 8000 |
| Centrifugo | Web Application | 17. Мгновенно доставляет WebSocket-пуш в открытую сессию рекрутера. Данные передаются по постоянному соединению с браузером. | WSS / JSON |
| Web Application | Рекрутер / Нанимающий менеджер | 18. Обновляет статус кандидата в воронке найма без перезагрузки страницы. | UI Render |
| Kafka | Hiring Service | 19. Автоматически переводит отклик в воронке на следующий этап. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro |
| Kafka | Notification Service | 20. Доставляет событие для формирования мультиканального уведомления. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro |
| Notification Service | Notification Providers | 21. Отправляет email кандидату с подтверждением сдачи. | HTTPS / REST API |

<!-- /excerpt -->

---

## 1. Архитектурный контекст сценария

Сценарий гармонично сочетает два потока обработки данных:
1. **Синхронный контур (Request-Response & Code Execution, Шаги 1–12)**: Прием решения кандидата в веб-интерфейсе, валидация сессии и дедлайна, проверка полномочий, запуск изолированного выполнения кода в Sandbox Runner, применение стратегии скоринга, транзакция изменения состояния сессии в PostgreSQL и синхронный возврат вердикта в UI соискателя.
2. **Асинхронный контур (Event-Driven & Realtime Push, Шаги 13–21)**: Непрерывный захват событий из журнала `PostgreSQL WAL` коннектором Debezium, доставка `AssessmentCompletedEvent` в шину Kafka, мгновенная трансляция push-уведомления рекрутерам через связку `Realtime Publisher` и `Centrifugo` (WebSocket), автоматическое продвижение отклика по воронке найма в `Hiring Service` и отправка email-уведомления соискателю.

В цепочке задействованы следующие контейнеры и узлы:
* **Web Application** (Next.js / React) — фронтенд-интерфейс соискателя (редактор кода и таймер) и нанимающей команды (канбан-доска и журнал откликов).
* **BFF (Backend for Frontend)** — серверная часть Next.js, проверяющая `HttpOnly`-куки, CSRF-токены и передающая токен доступа в API Gateway.
* **API Gateway** — шлюз платформы, обеспечивающий rate limiting, сквозную W3C-трассировку (`traceparent`) и инжекцию доверенного identity context.
* **Assessment Service** — микросервис управления библиотекой испытаний, сессиями кандидатов, скоринговыми стратегиями и фиксацией оценок на Go 1.23+.
* **OpenFGA** — движок авторизации на основе отношений (ReBAC), проверяющий права соискателя на сдачу решений в рамках активной сессии.
* **Code Execution Runner** — изолированная среда безопасной компиляции и исполнения решений кандидатов (Docker / gVisor runtime).
* **Assessment DB** (PostgreSQL 16) — реляционная СУБД, хранящая сессии, решения, результаты тестов и таблицу `outbox`.
* **Debezium Connector** — сервис Change Data Capture (CDC), считывающий журнал `PostgreSQL WAL`.
* **Apache Kafka (KRaft)** — распределённый брокер сообщений для гарантированной доставки доменных событий.
* **Realtime Publisher** — специализированный сервис трансляции доменных событий в пользовательские realtime-каналы.
* **Centrifugo** — масштабируемый WebSocket/SSE push-сервер, поддерживающий постоянные соединения с веб-клиентами.
* **Hiring Service** — микросервис найма, автоматически продвигающий заявку кандидата при успешном прохождении испытания.
* **Notification Service** — сервис мультиканальной рассылки оповещений.
* **Notification Providers** (SendGrid, FCM) — внешние платформы отправки email и push-уведомлений.

---

## 2. Подробная хронология и спецификация этапов (Шаги 1–21)

### Фаза А. Синхронный поток обработки запроса и исполнения (Шаги 1–8)

#### Шаг 1. Отправка решения задачи соискателем (Web Application)
* **Инициатор**: Кандидат (Соискатель) в онлайн-редакторе кода.
* **Получатель**: `Web Application` (Next.js).
* **Протокол и порт**: `HTTPS / TLS 1.3`, порт `443`.
* **Операция**: Клиентское действие отправки кода `submitSolution()`.
* **Входные данные (Payload)**:
  ```json
  {
    "sessionId": "b3c8a1d2-9876-4321-efab-0123456789de",
    "taskId": "f1e2d3c4-5678-90ab-cdef-1234567890ab",
    "language": "golang",
    "sourceCode": "package solution\n\nfunc TwoSum(nums []int, target int) []int {\n\tm := make(map[int]int)\n\tfor i, n := range nums {\n\t\tif j, ok := m[target-n]; ok { return []int{j, i} }\n\t\tm[n] = i\n\t}\n\treturn nil\n}"
  }
  ```

#### Шаг 2. Проксирование вызова через сессионный слой (BFF)
* **Инициатор**: `Web Application`.
* **Получатель**: `BFF (Backend for Frontend)`.
* **Протокол и порт**: `HTTPS / JSON`, порт `443` (внутренняя сеть).
* **Операция**: `POST /api/assessment/sessions/{sessionId}/submit`.
* **Логика**: BFF валидирует сессионную куку `__Host-Careeria-Session`, извлекает CSRF-токен из заголовка `X-CSRF-Token`, обогащает вызов токеном доступа и формирует HTTP-запрос к шлюзу.

#### Шаг 3. Авторизованный вызов шлюза платформы (API Gateway)
* **Инициатор**: `BFF`.
* **Получатель**: `API Gateway`.
* **Протокол и порт**: `HTTP / JSON`, порт `8080`.
* **Операция**: Отправка решения через HTTP API с токеном доступа.
* **Логика**: API Gateway проверяет rate limit по IP и пользователю, генерирует `X-Request-ID`, считывает трассировочные заголовки W3C Trace Context и передаёт идентификатор аутентифицированного субъекта в метаданных gRPC.

#### Шаг 4. Вызов RPC-метода Assessment Service
* **Инициатор**: `API Gateway`.
* **Получатель**: `Assessment Service` (`gRPC Server`).
* **Протокол и порт**: `gRPC / HTTP2`, сетевой порт `:50055`.
* **Операция**: `careeria.assessment.v1.AssessmentService/SubmitTaskSolution`.
* **Контракт вызова (Protobuf DTO)**:
  ```protobuf
  message SubmitTaskSolutionRequest {
    string session_id = 1 [(validate.rules).string.uuid = true];
    string task_id = 2 [(validate.rules).string.uuid = true];
    string language = 3 [(validate.rules).string.min_len = 2];
    string source_code = 4 [(validate.rules).string.min_len = 10];
  }
  ```

#### Шаг 5. Валидация прав и политик сессии (OpenFGA & Domain Policy)
* **Инициатор**: `Assessment Service` (`Session Execution Service` / `Authorization Client`).
* **Получатель**: `OpenFGA`.
* **Протокол и порт**: `gRPC / mTLS`, сетевой порт `:8081`.
* **Операция**: `openfga.v1.OpenFGAService/Check`.
* **Контракт вызова**: `user: candidate:<UUID>, relation: "can_submit", object: "assessment_session:<UUID>"`.
* **Логика**: Проверяется владение сессией. Дополнительно `SessionLifecyclePolicy` в чистом доменном коде удостоверяет: сессия находится в статусе `IN_PROGRESS`, дедлайн `expires_at` не наступил, и язык программирования присутствует в белом списке шаблона (`allowed_languages`).

#### Шаг 6. Направление кода на изолированное исполнение (Code Execution Runner)
* **Инициатор**: `Assessment Service` (`Sandbox Client`).
* **Получатель**: `Code Execution Runner`.
* **Протокол и порт**: `gRPC / HTTP2`, сетевой порт `:50060`.
* **Операция**: `careeria.runner.v1.RunnerService/ExecuteCode`.
* **Логика**: Раннер разворачивает изолированный sandbox-контейнер (gVisor/Docker), монтирует закрытый набор тест-кейсов (включая скрытые бенчмарки и краевые тесты), компилирует код кандидата и исполняет юнит-тесты с жесткими квотами (лимит времени 5 сек, память 256 МБ, запрет сетевых вызовов).

#### Шаг 7. Возврат протокола тестирования и метрик
* **Инициатор**: `Code Execution Runner`.
* **Получатель**: `Assessment Service`.
* **Протокол и порт**: `gRPC response`.
* **Ответ (Protobuf DTO)**:
  ```protobuf
  message ExecuteCodeResponse {
    enum Status { PASSED = 0; FAILED = 1; COMPILATION_ERROR = 2; TIMEOUT = 3; }
    Status status = 1;
    int32 passed_tests = 2;
    int32 total_tests = 3;
    int64 execution_time_ms = 4;
    int64 memory_used_bytes = 5;
    string compiler_output = 6;
  }
  ```

#### Шаг 8. Скоринг, FSM-переход и транзакция в Assessment DB
* **Инициатор**: `Assessment Service` (`Scoring Context & Strategies`).
* **Получатель**: `Assessment DB` (PostgreSQL 16).
* **Протокол и порт**: `SQL / TCP`, порт `:5432`.
* **Операция**: Физическая транзакция `BEGIN ... COMMIT`.
* **Внутренняя логика**:
  1. Вызывается доменный контекст скоринга `ScoringContext` с реализацией стратегии `WeightedScoringStrategy` (или `StrictAutoScoringStrategy`).
  2. Вычисляется итоговый балл: например, `final_score = 95.0`, флаг прохождения `is_passed = true` (превышает `passing_threshold = 75.0`).
  3. Сохраняется запись попытки в таблицу `task_submissions` со статусом `PASSED` и метриками исполнения.
  4. Конечный автомат агрегата `AssessmentSession` атомарно переходит из `IN_PROGRESS` в терминальное состояние `EVALUATED`.
  5. В служебную таблицу `assessment_outbox` сериализуется событие `AssessmentCompletedEvent`.
  6. Выполняется `COMMIT`.

---

### Фаза Б. Возврат синхронного ответа кандидату (Шаги 9–12)

#### Шаг 9. Возврат статуса операции из Assessment Service
* **Инициатор**: `Assessment Service`.
* **Получатель**: `API Gateway`.
* **Протокол и порт**: `gRPC / HTTP2`, порт `:50055`.
* **Ответ (Protobuf Response)**:
  ```protobuf
  message SubmitTaskSolutionResponse {
    string session_id = 1;
    string status = 2; // "EVALUATED"
    double final_score = 3; // 95.0
    bool is_passed = 4; // true
    int32 passed_tests = 5;
    int32 total_tests = 6;
  }
  ```

#### Шаг 10. Трансляция ответа шлюзом платформы
* **Инициатор**: `API Gateway`.
* **Получатель**: `BFF`.
* **Протокол и порт**: HTTP / JSON, порт 8080.
* **Логика**: Шлюз закрывает OpenTelemetry span, валидирует статус `codes.OK` и пересылает структурированный ответ в BFF.

#### Шаг 11. Формирование HTTP-ответа клиенту (BFF)
* **Инициатор**: `BFF`.
* **Получатель**: `Web Application`.
* **Протокол и порт**: `HTTPS / JSON`, статус `200 OK`.
* **Тело ответа (JSON Payload)**:
  ```json
  {
    "success": true,
    "sessionId": "b3c8a1d2-9876-4321-efab-0123456789de",
    "status": "EVALUATED",
    "finalScore": 95.0,
    "isPassed": true,
    "message": "Решение успешно протестировано. Задание выполнено на отлично!"
  }
  ```

#### Шаг 12. Отображение вердикта в интерфейсе соискателя
* **Инициатор**: `Web Application` (браузер соискателя).
* **Получатель**: Соискатель (Кандидат).
* **Логика**: Интерфейс деактивирует поле ввода кода, останавливает таймер испытания, выводит плашку об успешной сдаче с итоговым баллом 95% и рекомендацией ожидать связи с рекрутером.

---

### Фаза В. Асинхронная доставка событий, Realtime и нотификации (Шаги 13–21)

#### Шаг 13. Захват события из журнала WAL (Debezium CDC)
* **Инициатор**: `Assessment DB` (PostgreSQL WAL).
* **Получатель**: `Debezium Connector`.
* **Протокол и порт**: `pgoutput Logical Replication`, порт `:5432`.
* **Логика**: Debezium считывает зафиксированную строку события `AssessmentCompletedEvent` из слота репликации `careeria_assessment_slot` в режиме реального времени с нулевыми потерями.

#### Шаг 14. Публикация доменного события в Apache Kafka
* **Инициатор**: `Debezium Connector`.
* **Получатель**: `Apache Kafka` (топик `careeria.assessment.events`).
* **Протокол и порт**: `Kafka Protocol / TCP`, порт `:9092`.
* **Формат данных**: Бинарный Avro payload, валидированный через Apicurio Schema Registry, с подтверждением записи `acks=all`.

#### Шаг 15. Вычитывание события сервисом Realtime Publisher
* **Инициатор**: `Apache Kafka`.
* **Получатель**: `Realtime Publisher`.
* **Протокол и порт**: `Kafka Protocol / TCP`, порт `:9092`.
* **Логика**: Сервис вычитывает событие в консьюмер-группе `careeria.realtime.assessment`, определяет целевой канал рекрутеров компании `company:recruiting:<companyId>` и формирует JSON-сообщение.

#### Шаг 16. Публикация события в Centrifugo API
* **Инициатор**: `Realtime Publisher`.
* **Получатель**: `Centrifugo Server`.
* **Протокол и порт**: `Centrifugo HTTP API / JSON`, порт `:8000`.
* **Операция**: `POST /api/publish`.
* **Полезная нагрузка**:
  ```json
  {
    "channel": "company:recruiting:456",
    "data": {
      "type": "ASSESSMENT_COMPLETED",
      "candidateId": "c1a2b3c4-...",
      "applicationId": "d4e5f6a7-...",
      "score": 95.0,
      "isPassed": true
    }
  }
  ```

#### Шаг 17. Доставка WebSocket-сообщения рекрутеру
* **Инициатор**: `Centrifugo Server`.
* **Получатель**: `Web Application` (браузер рекрутера/интервьюера).
* **Протокол и порт**: `WebSocket (WSS) / JSON`, порт `:443`.
* **Логика**: Centrifugo отправляет фрейм через постоянное WebSocket-соединение клиентам, подписанным на канал компании.

#### Шаг 18. Обновление UI рекрутера в реальном времени
* **Инициатор**: `Web Application` (браузер рекрутера).
* **Получатель**: Рекрутер / Нанимающий менеджер.
* **Логика**: TanStack Query на фронтенде обновляет состояние карточки кандидата в воронке найма без перезагрузки веб-страницы: загорается зеленый бейдж `Тест пройден: 95%`.

#### Шаг 19. Автоматическое продвижение в воронке найма (Hiring Service)
* **Инициатор**: `Apache Kafka`.
* **Получатель**: `Hiring Service`.
* **Протокол и порт**: `Kafka Consumer Protocol / TCP`, порт `:9092`.
* **Логика**: `Hiring Service` считывает событие `AssessmentCompletedEvent`. Так как `is_passed == true`, сервис инициирует автоматический переход заявки со стадии `TECHNICAL_TEST` на этап `TECH_INTERVIEW_PENDING` с записью в аудит-лог.

#### Шаг 20. Формирование оповещения (Notification Service)
* **Инициатор**: `Apache Kafka`.
* **Получатель**: `Notification Service`.
* **Протокол и порт**: `Kafka Consumer Protocol / TCP`, порт `:9092`.
* **Логика**: Сервис подготавливает персонализированный email-шаблон для соискателя с официальным подтверждением успешного завершения испытания.

#### Шаг 21. Отправка подтверждения соискателю (Notification Providers)
* **Инициатор**: `Notification Service`.
* **Получатель**: `Notification Providers` (SendGrid / FCM).
* **Протокол и порт**: `HTTPS / REST API`, порт `:443`.
* **Логика**: Кандидату доставляется транзакционное email-письмо с итоговым протоколом прохождения тестирования.

---

## 3. Сводная спецификация параметров динамического процесса

<p align="right">Таблица - Сводная спецификация параметров динамического процесса (Сквозной динамический процесс оценки компетенций и realtime-оповещения (Dynamic Diagram))</p>

| № шага | Направление | Инициатор вызова | Целевой элемент | Вызываемый интерфейс / операция | Протокол / Порт | Формат данных | Тип взаимодействия | SLA / Таймаут |
|:---:|:---:|---|---|---|:---:|---|:---:|:---:|
| **1** | Запрос | Кандидат | Web Application | `submitSolution()` | HTTPS / 443 | JSON DTO | Синхронный | < 50 мс |
| **2** | Запрос | Web Application | BFF | `POST /api/assessment/sessions/{id}/submit` | HTTPS / 443 | JSON + Cookie | Синхронный | < 100 мс |
| **3** | Запрос | BFF | API Gateway | Отправка решения | HTTP / 8080 | JSON | Синхронный | < 150 мс |
| **4** | Запрос | API Gateway | Assessment Service | `AssessmentService/SubmitTaskSolution` | gRPC / 50055 | Protobuf DTO | Синхронный | < 200 мс |
| **5** | Запрос | Assessment Service | OpenFGA | `OpenFGAService/Check` | gRPC / 8081 | ReBAC Tuple | Синхронный | < 20 мс |
| **6** | Запрос | Assessment Service | Code Execution Runner | `RunnerService/ExecuteCode` | gRPC / 50060 | Protobuf (Source + Tests) | Синхронный | < 3000 мс |
| **7** | Ответ | Code Execution Runner | Assessment Service | `ExecuteCodeResponse` | gRPC / 50060 | Protobuf (Report & Metrics) | Синхронный | < 50 мс |
| **8** | Запрос | Assessment Service | Assessment DB | `SQL Transaction (BEGIN ... COMMIT)` | SQL / 5432 | SQL Statements | Синхронный | < 50 мс |
| **9** | **Ответ** | **Assessment Service** | **API Gateway** | **`SubmitTaskSolutionResponse`** | **gRPC / 50055** | **Protobuf Response** | **Синхронный** | **< 30 мс** |
| **10** | **Ответ** | **API Gateway** | **BFF** | Ответ шлюза | HTTP / 8080 | JSON | **Синхронный** | **< 30 мс** |
| **11** | **Ответ** | **BFF** | **Web Application** | **`HTTP 200 OK (JSON Verdict)`** | **HTTPS / 443** | **JSON Payload** | **Синхронный** | **< 50 мс** |
| **12** | **Ответ** | **Web Application** | **Кандидат** | **Рендеринг UI (Score, Results, Toast)** | **UI Event** | **DOM / React State** | **Клиентский** | **< 16 мс** |
| **13** | Событие | Assessment DB | Debezium Connector | Потоковое чтение PostgreSQL WAL | WAL / 5432 | Бинарный pgoutput WAL | Асинхронный (CDC) | < 100 мс |
| **14** | Событие | Debezium Connector | Apache Kafka | `ProduceRecord(assessment.events)` | Kafka / 9092 | Apache Avro Binary | Асинхронный | < 50 мс |
| **15** | Событие | Apache Kafka | Realtime Publisher | `FetchRecord(assessment.events)` | Kafka / 9092 | Apache Avro Binary | Асинхронный | < 50 мс |
| **16** | Вызов | Realtime Publisher | Centrifugo Server | `POST /api/publish` | HTTP / 8000 | JSON Payload | Асинхронный | < 30 мс |
| **17** | Push | Centrifugo Server | Web Application (Рекрутер) | WSS Broadcast Event | WSS / 443 | WebSocket Frame | Realtime Push | < 20 мс |
| **18** | UI | Web Application | Рекрутер / Интервьюер | Обновление карточки кандидата | UI Event | DOM / TanStack Query | Клиентский | < 16 мс |
| **19** | Событие | Apache Kafka | Hiring Service | `FetchRecord(assessment.events)` | Kafka / 9092 | Apache Avro Binary | Асинхронный | < 50 мс |
| **20** | Событие | Apache Kafka | Notification Service | `FetchRecord(assessment.events)` | Kafka / 9092 | Apache Avro Binary | Асинхронный | < 50 мс |
| **21** | Внешний | Notification Service | Notification Providers | `SendTransactionalEmail` | HTTPS / 443 | JSON Payload | Асинхронный | < 1.5 с |

---

## 4. Граничные сценарии и матрица обработки ошибок

<p align="right">Таблица - Граничные сценарии и матрица обработки ошибок (Сквозной динамический процесс оценки компетенций и realtime-оповещения (Dynamic Diagram))</p>

| Возникающая исключительная ситуация | Слой обнаружения | Доменная ошибка | HTTP / gRPC код ответа | Архитектурная реакция и компенсация |
|---|---|---|---|---|
| Истекло время сдачи испытания (`NOW() > expires_at`) | Domain Policy (`SessionLifecyclePolicy`) | `ErrSessionExpired` | `410 Gone` / `codes.FailedPrecondition` | Прерывание обработки. Перевод FSM сессии в статус `EXPIRED`. Решение не передается в раннер. |
| Попытка сдачи от чужого имени или неавторизованного пользователя | OpenFGA / Authz Client | `ErrUnauthorizedCandidate` | `403 Forbidden` / `codes.PermissionDenied` | Блокировка вызова, фиксация инцидента безопасности в логе аудита. |
| Ошибка компиляции кода соискателя (`syntax error`, `unresolved import`) | Code Execution Runner | `ErrCompilationFailed` | `200 OK` / `codes.OK` (с отчетом) | Решение сохраняется в `task_submissions` со статусом `COMPILATION_ERROR`. Ошибки компилятора возвращаются соискателю для исправления. |
| Превышение лимита времени выполнения кода в песочнице (> 5000 мс) | Code Execution Runner | `ErrExecutionTimeout` | `200 OK` / `codes.OK` (с отчетом) | Контейнер песочницы принудительно уничтожается (`SIGKILL`). Решение помечается статусом `TIMEOUT`, баллы за тест не начисляются. |
| Временная недоступность пула раннеров кода (Runner Out of Capacity) | Sandbox Client (Circuit Breaker) | `ErrRunnerUnavailable` | `503 Service Unavailable` / `codes.Unavailable` | Срабатывание Circuit Breaker. Запрос отклоняется с предложением повторить отправку через несколько секунд без списания времени таймера. |
| Сбой сетевой доставки WebSocket через Centrifugo | Centrifugo / Web Client | `ErrWebSocketDisconnected` | Ошибка транспортного уровня WS | Клиентский SDK Centrifugo автоматически переподключается с экспоненциальным backoff и докачивает пропущенные события из истории канала по `offset`. |
| Сбой отправки письма через внешнего провайдера (SendGrid) | Notification Service | `ErrNotificationProviderTimeout` | Внутренняя ошибка консьюмера | Сообщение в Kafka не коммитится, отправляется в retry-топик с экспоненциальной задержкой. Основной бизнес-процесс оценки не блокируется. |

---

[Ландшафт L0](../members/member2/landscape/member2-l0-landscape.md) · [Контекст L1](../members/member3/context%28L1%29/index.md) · [Контейнеры L2](../members/member3/context%28L1%29/containers%28L2%29/index.md) · [Развёртывание подсистемы (L2)](../members/member3/deployment/member3-deployment.md) · [Раздел Member 3](../members/member3/README.md)

