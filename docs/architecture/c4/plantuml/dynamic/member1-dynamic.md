# Сквозной динамический процесс публикации вакансии (Dynamic Diagram)

<!-- excerpt -->

Рекрутер публикует вакансию. Vacancy Service проверяет полномочия и сохраняет новый статус, после чего подписчики события обновляют права на просмотр и запускают рассылку.

Команда проходит через серверный BFF и API Gateway в Vacancy Service. После проверки прав и готовности вакансии сервис сохраняет состояние публикации и outbox в одной транзакции. Debezium передаёт событие в Kafka. Обработчик синхронизации доступа обновляет отношения OpenFGA, сервис уведомлений организует рассылку.

```plantuml
@startuml
!include architecture/c4/plantuml/dynamic/member1-dynamic.puml
@enduml
```

<p align="center">Рисунок - Публикация вакансии</p>

<p align="right">Таблица - Элементы. Публикация вакансии</p>

| Элемент | Тип или технология | Назначение и ответственность |
|---|---|---|
| Рекрутер | Пользователь | Редактирует черновик, инициирует публикацию вакансии и получает подтверждение в UI. |
| Web Application | Next.js, React, TypeScript, Tailwind CSS | Интерфейс React работает в браузере. Серверный BFF обрабатывает пользовательскую сессию и обращается к API Gateway. Страницы и BFF развёртываются вместе в одном приложении Next.js. |
| API Gateway | Go, grpc-gateway | API Gateway принимает HTTP-запрос от BFF, проверяет предъявленный токен по открытым ключам Identity и применяет политику доступа к методу. Доменный сервис получает вызов по gRPC. |
| Vacancy Service | Go, gRPC, Protobuf, pgx, sqlc | Vacancy Service хранит содержание и условия вакансии. При публикации сервис проверяет права рекрутера и готовность вакансии, затем фиксирует её новый статус в Vacancy DB. |
| OpenFGA | OpenFGA, PostgreSQL | Сервис проверки доступа на основе отношений между субъектом и ресурсом. Доменные сервисы запрашивают решение о доступе, а обработчик синхронизации обновляет отношения по событиям платформы. |
| Vacancy DB | PostgreSQL | Вакансии, навыки и transactional outbox. |
| Debezium | Debezium PostgreSQL Connector | Компонент захвата зафиксированных изменений PostgreSQL. Читает transactional outbox через журнал WAL и публикует доменные события в Kafka. |
| Kafka | Apache Kafka, KRaft | Событийная шина платформы. Хранит доменные сообщения в топиках и передаёт их группам потребителей для независимой асинхронной обработки. |
| AuthZ Synchronization Service | Go, Kafka Consumer, OpenFGA SDK | AuthZ Synchronization Service читает доменные события из Kafka и обновляет отношения доступа в OpenFGA. Например, после публикации вакансии сервис добавляет право на чтение её карточки. |
| Notification Service | Go, Kafka Consumer | Notification Service принимает команды и события из Kafka, выбирает шаблон и создаёт задачу доставки. Обработчики сохраняют уведомления в личном кабинете или передают их провайдерам email, push и SMS. |
| Notification Providers | Система | Внешние поставщики доставки сообщений. SendGrid обрабатывает email, Firebase Cloud Messaging доставляет push, Twilio отправляет SMS по запросам сервиса уведомлений. |

<p align="right">Таблица - Связи. Публикация вакансии</p>

| Источник | Получатель | Содержание взаимодействия | Протокол или способ вызова |
|---|---|---|---|
| Рекрутер | Web Application | 1-2. Отправляет форму публикации с сессионной cookie. | HTTPS / UI |
| Web Application | API Gateway | 3. BFF передаёт авторизованный запрос. | HTTP / JSON |
| API Gateway | Vacancy Service | 4. Вызывает метод PublishVacancy(). Запрос и ответ передаются как сообщения Protobuf. | gRPC / HTTP2, порт 50054 |
| Vacancy Service | OpenFGA | 5. Проверяет права публикатора (can_publish). Запрос и ответ передаются как сообщения Protobuf. | gRPC / mTLS, порт 8081 |
| Vacancy Service | Vacancy DB | 6. Фиксирует статус PUBLISHED (после PublishingPolicy) и пишет Outbox. Получатель выполняет чтение или запись в базе данных сервиса. | SQL Transaction, порт 5432 |
| Vacancy Service | API Gateway | 7. Возвращает статус OK и DTO вакансии. Запрос и ответ передаются как сообщения Protobuf. | gRPC response |
| API Gateway | Web Application | 8. Возвращает ответ серверному BFF. | HTTP / JSON |
| Web Application | Рекрутер | 9-10. Возвращает и отображает статус вакансии. | UI Render |
| Vacancy DB | Debezium | 11. Считывает запись Outbox из журнала транзакций WAL (CDC). Коннектор читает зафиксированные изменения из журнала PostgreSQL. | PostgreSQL WAL (pgoutput) |
| Debezium | Kafka | 12. Публикует VacancyPublished в топик careeria.vacancies.events.v1. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro, порт 9092 |
| Kafka | AuthZ Synchronization Service | 13. Доставляет событие публикации вакансии. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro |
| AuthZ Synchronization Service | OpenFGA | 14. Записывает ReBAC-правила публичного доступа к вакансии. Запрос и ответ передаются как сообщения Protobuf. | OpenFGA API / gRPC, порт 8081 |
| Kafka | Notification Service | 15. Вычитывает событие публикации вакансии. Сообщение передаётся асинхронно через топик Kafka. | Kafka Protocol / TCP |
| Notification Service | Notification Providers | 16. Инициирует рассылку уведомлений релевантным кандидатам. | HTTPS / REST API |

<!-- /excerpt -->

---

## 1. Архитектурный контекст сценария

В отличие от многоступенчатых процессов с промежуточными клиентами, диаграмма демонстрирует **один цельный сквозной бизнес-процесс** от единственного действия пользователя до завершения всех синхронных и асинхронных реакций системы:
1. **Синхронный контур (Request-Response, Шаги 1–10)**: Обработка команды рекрутера в веб-интерфейсе, проверка ReBAC-полномочий, проверка доменной политики `PublishingPolicy`, транзакционный перевод конечного автомата в БД и обязательный возврат актуального статуса в UI.
2. **Асинхронный контур (Event-Driven, Шаги 11–16)**: Фоновая доставка доменного события `VacancyPublished` через Transactional Outbox, CDC Debezium и шину Kafka с обновлением проекции публичной видимости в `OpenFGA` и оповещением релевантных соискателей через `Notification Service`.

В цепочке задействованы следующие контейнеры и узлы:
* **Web Application** (Next.js / React) — фронтенд-интерфейс рекрутера.
* **BFF (Backend for Frontend)** — серверная часть Next.js, проверяющая `HttpOnly`-куки и авторизационные заголовки.
* **API Gateway** — шлюз платформы, обеспечивающий rate limiting, распределённую трассировку W3C и передачу доверенного identity context.
* **Vacancy Service** — микросервис бизнес-логики вакансий на Go 1.23+, управляющий агрегатом `Vacancy`, конечным автоматом FSM и политикой публикации `PublishingPolicy`.
* **OpenFGA** — специализированный движок авторизации на основе отношений (ReBAC).
* **Vacancy DB** (PostgreSQL 16) — логическая база данных `vacancy_db`, хранящая таблицы `vacancies`, `vacancy_skills` и `outbox_events`.
* **Debezium Connector** — сервис захвата изменений данных (Change Data Capture, CDC), считывающий журнал `PostgreSQL WAL`.
* **Apache Kafka (KRaft)** — распределённый брокер сообщений, обеспечивающий надёжную доставку доменных событий в топике `careeria.vacancies.events.v1`.
* **AuthZ Synchronization Service** — фоновый Kafka-консьюмер, обновляющий ReBAC-отношения доступа в `OpenFGA` по доменным событиям.
* **Notification Service** — сервис мультиканальной рассылки оповещений.
* **Notification Providers** (SendGrid, FCM, Twilio) — внешние коммуникационные платформы доставки уведомлений кандидатам.

---

## 2. Подробная хронология и спецификация этапов (Шаги 1–16)

### Фаза А. Синхронный поток обработки запроса (Шаги 1–6)

#### Шаг 1. Инициация публикации вакансии (`Web Application`)
* **Инициатор**: Рекрутер (HR-специалист компании).
* **Получатель**: `Web Application` (Next.js).
* **Протокол и порт**: `HTTPS / TLS 1.3`, порт `443`.
* **Операция**: Пользователь открывает карточку черновика вакансии (`DRAFT`) и нажимает кнопку «Опубликовать вакансию» (`publishVacancy()`).

#### Шаг 2. Передача команды на сессионный фасад (`BFF`)
* **Инициатор**: `Web Application`.
* **Получатель**: серверный BFF приложения Next.js.
* **Протокол и порт**: `HTTPS / JSON`, порт `443`.
* **Операция**: `POST /api/vacancies/{vacancyId}/publish`. `BFF` валидирует защищённую cookie `__Host-Careeria-Session`, проверяет `X-CSRF-Token` и извлекает идентификатор пользователя из сессии.

#### Шаг 3. Авторизованный вызов шлюза (`API Gateway`)
* **Инициатор**: `BFF`.
* **Получатель**: `API Gateway` (`grpc-gateway`).
* **Протокол и порт**: `HTTP / JSON`, порт `8080`.
* **Операция**: BFF передаёт шлюзу команду публикации вакансии с токеном доступа в HTTP-заголовке Authorization.

#### Шаг 4. Маршрутизация команды в домен вакансий (`Vacancy Service`)
* **Инициатор**: `API Gateway`.
* **Получатель**: `Vacancy Service`.
* **Протокол и порт**: `gRPC / HTTP2`, порт `50054`.
* **Операция**: `VacancyService/PublishVacancy`. Внутри контейнера компонент `gRPC Server` проверяет структуру Protobuf-запроса и передаёт управление прикладному компоненту `Vacancy Service`.
* **Входные данные (Payload)**:
  ```json
  {
    "vacancyId": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "expectedVersion": 3
  }
  ```

#### Шаг 5. Проверка ReBAC-полномочий (`OpenFGA`)
* **Инициатор**: `Vacancy Service` (компонент `Authorization Client`).
* **Получатель**: `OpenFGA`.
* **Протокол и порт**: `gRPC / mTLS`, порт `8081`.
* **Операция**: Проверка отношения `check(user:<userId>, relation: "can_publish", object: "vacancy:<vacancyId>")` (с учётом родительской компании). При отсутствии прав выполнение немедленно прерывается с кодом `codes.PermissionDenied`.

#### Шаг 6. Валидация политики, переход FSM и запись Outbox (`Vacancy DB`)
* **Инициатор**: `Vacancy Service` (компоненты `Vacancy Repo` и `Transactional Outbox`).
* **Получатель**: `Vacancy DB` (логическая база `vacancy_db` в PostgreSQL 16).
* **Протокол и порт**: `SQL Transaction / TCP`, порт `5432`.
* **Выполнение в единой ACID-транзакции (`BEGIN ... COMMIT`)**:
  1. Агрегат `Vacancy` загружается вместе со списком требуемых навыков (`vacancy_skills`).
  2. Доменный метод `Vacancy.Publish()` проверяет допустимость перехода конечного автомата (`DRAFT` $\rightarrow$ `PUBLISHED` или `ARCHIVED` $\rightarrow$ `PUBLISHED`; статус `CLOSED` отклоняется) и применяет политику готовности **`PublishingPolicy`**:
     * заголовок `title` длиной не менее 5 символов;
     * описание `description` длиной не менее 50 символов;
     * наличие хотя бы одного привязанного навыка в `vacancy_skills`.
  3. Выполняется обновление строки в `vacancies` (`status = 'PUBLISHED'`, `published_at = NOW()`, инкремент версии) с контролем оптимистичной блокировки.
  4. Через компонент `Transactional Outbox` в таблицу `outbox_events` атомарно сохраняется событие `VacancyPublished` (`eventType: "careeria.vacancy.published"`).

---

### Фаза Б. Синхронный возврат результата клиенту (Шаги 7–10)

* **Шаг 7 (`gRPC response`, порт 50054):** `Vacancy Service` возвращает `PublishVacancyResponse` (идентификатор, статус `PUBLISHED`, `publishedAt`, обновлённую версию) в `API Gateway`.
* **Шаг 8 (`HTTP / JSON`, порт 8080):** `API Gateway` транслирует ответ в `BFF`.
* **Шаг 9 (`HTTPS / JSON`, порт 443):** `BFF` возвращает `200 OK` в браузер пользователя.
* **Шаг 10 (`UI Render`):** `Web Application` обновляет состояние интерфейса (TanStack Query), переводя бейдж вакансии в состояние «Опубликована» и отображая уведомление об успехе.

---

### Фаза В. Асинхронный поток: CDC, Kafka и подписчики (Шаги 11–16)

* **Шаг 11 (`PostgreSQL WAL`, порт 5432):** `Debezium Connector` непрерывно считывает новую запись из таблицы `outbox_events` логической базы `vacancy_db` через слот логической репликации `careeria_vacancy_slot` (`pgoutput`) без polling-запросов к БД.
* **Шаг 12 (`Kafka Protocol / TCP`, порт 9092):** `Debezium` сериализует событие `VacancyPublished` в бинарный формат Apache Avro (валидация схемы через Apicurio Schema Registry) и публикует его в топик **`careeria.vacancies.events.v1`** (`acks=all`, `replication.factor=3`).
* **Шаги 13–14 (`Kafka → OpenFGA API`, порт 8081):** Платформенный воркер `AuthZ Synchronization Service` вычитывает событие `VacancyPublished` и записывает в `OpenFGA` ReBAC-кортежи публичного чтения опубликованной вакансии (открывая доступ к карточке для соискателей без предоставления доступа к внутренним черновикам).
* **Шаги 15–16 (`Kafka → HTTPS / REST API`, порт 443):** Микросервис `Notification Service` вычитывает событие в консьюмер-группе `careeria.notifications.vacancies` (с проверкой идемпотентности по таблице `inbox`) и передаёт команды во внешние шлюзы `Notification Providers` (`SendGrid` / `FCM`) для оповещения кандидатов с релевантными навыками о новой открытой вакансии.

---

## 3. Сводная спецификация параметров динамического процесса

<p align="right">Таблица - Сводная спецификация параметров динамического процесса (Сквозной динамический процесс публикации вакансии (Dynamic Diagram))</p>

| № | Направление | Инициатор | Получатель | Операция / интерфейс | Протокол / Порт | Формат данных | Тип вызова | SLA / Таймаут |
|:---:|---|---|---|---|---|---|---|---|
| **1** | Запрос | Рекрутер | Web Application | `publishVacancy()` | HTTPS / 443 | UI Action | Клиентский | < 50 мс |
| **2** | Запрос | Web Application | BFF | `POST /api/vacancies/{id}/publish` | HTTPS / 443 | JSON + Cookie | Синхронный | < 100 мс |
| **3** | Запрос | BFF | API Gateway | Публикация вакансии | HTTP / 8080 | JSON | Синхронный | < 120 мс |
| **4** | Запрос | API Gateway | Vacancy Service | `VacancyService/PublishVacancy` | gRPC / 50054 | Protobuf DTO | Синхронный | < 200 мс |
| **5** | Запрос | Vacancy Service | OpenFGA | `OpenFGAService/Check(can_publish)` | gRPC / 8081 | Protobuf ReBAC Tuple | Синхронный | < 20 мс |
| **6** | Запрос | Vacancy Service | Vacancy DB | `SQL Transaction (PUBLISHED + Outbox)` | SQL / 5432 | SQL Statements | Синхронный | < 50 мс |
| **7** | **Ответ** | **Vacancy Service** | **API Gateway** | **`PublishVacancyResponse`** | **gRPC / 50054** | **Protobuf Response** | **Синхронный** | **< 20 мс** |
| **8** | **Ответ** | **API Gateway** | **BFF** | Ответ шлюза | HTTP / 8080 | JSON | **Синхронный** | **< 20 мс** |
| **9** | **Ответ** | **BFF** | **Web Application** | **`HTTP 200 OK (JSON Status)`** | **HTTPS / 443** | **JSON Payload** | **Синхронный** | **< 40 мс** |
| **10** | **Ответ** | **Web Application** | **Рекрутер** | **Рендеринг UI (статус PUBLISHED)** | **UI Event** | **DOM / React State** | **Клиентский** | **< 16 мс** |
| **11** | Событие | Vacancy DB | Debezium Connector | Потоковое чтение PostgreSQL WAL | WAL / 5432 | Бинарный `pgoutput` WAL | Асинхронный (CDC) | < 100 мс |
| **12** | Событие | Debezium Connector | Apache Kafka | `ProduceRecord(careeria.vacancies.events.v1)` | Kafka / 9092 | Apache Avro Binary | Асинхронный | < 50 мс |
| **13** | Событие | Apache Kafka | AuthZ Synchronization | `FetchRecord(careeria.vacancies.events.v1)` | Kafka / 9092 | Apache Avro Binary | Асинхронный | < 50 мс |
| **14** | Запрос | AuthZ Synchronization | OpenFGA | `WriteTuples(vacancy_public_view)` | gRPC / 8081 | Protobuf ReBAC Tuple | Асинхронный | < 30 мс |
| **15** | Событие | Apache Kafka | Notification Service | `FetchRecord(careeria.vacancies.events.v1)` | Kafka / 9092 | Apache Avro Binary | Асинхронный | < 50 мс |
| **16** | Событие | Notification Service | Notification Providers | `SendVacancyAlertEmail / Push` | HTTPS / 443 | JSON Payload | Асинхронный | < 1.5 с |

---

## 4. Граничные сценарии и матрица обработки ошибок

<p align="right">Таблица - Граничные сценарии и матрица обработки ошибок (Сквозной динамический процесс публикации вакансии (Dynamic Diagram))</p>

| Исключительная ситуация | Слой обнаружения | Доменная ошибка | HTTP / gRPC код ответа | Архитектурная реакция и компенсация |
|---|---|---|---|---|
| Рекрутер не принадлежит компании или не имеет права публикации | OpenFGA / Authorization Client (шаг 5) | `ErrUnauthorizedPublish` | `403 Forbidden` / `codes.PermissionDenied` | Прерывание операции до обращения к `Vacancy DB`. Логирование попытки несанкционированного доступа в аудит. |
| Черновик не соответствует `PublishingPolicy` (`title < 5`, `description < 50` или 0 навыков) | Domain Policy (`PublishingPolicy`, шаг 6) | `ErrVacancyPolicyViolated` | `422 Unprocessable Entity` / `codes.FailedPrecondition` | Отклонение перехода без записи в `outbox_events`. Возврат списка незаполненных требований для подсказки в UI. |
| Попытка повторно опубликовать уже закрытую вакансию (`status == CLOSED`) | Domain Aggregate (`Vacancy.Publish`, шаг 6) | `ErrInvalidVacancyTransition` | `409 Conflict` / `codes.FailedPrecondition` | Запрещено инвариантом FSM (публикация возможна только из `DRAFT` или `ARCHIVED`). Откат транзакции. |
| Конкурентное редактирование черновика другим рекрутером (`version mismatch`) | Infrastructure (`pgx`/`sqlc`, шаг 6) | `ErrOptimisticLockConflict` | `409 Conflict` / `codes.Aborted` | Откат SQL-транзакции (`ROLLBACK`). Клиенту возвращается предложение обновить карточку вакансии и повторить публикацию. |
| Сбой внешней доставки уведомлений кандидатам через SendGrid / FCM | Notification Service (шаг 16) | `ErrExternalGatewayTimeout` | Ошибка консьюмера | Offset в Kafka не коммитится, событие направляется в очередь повторной доставки (`Retry Topic` → `DLQ`) с защитой от дублей в `inbox`. |

---

[Ландшафт L0](../members/member1/landscape/member1-l0-landscape.md) · [Контекст L1](../members/member1/context%28L1%29/index.md) · [Контейнеры L2](../members/member1/context%28L1%29/containers%28L2%29/index.md) · [Развёртывание](../members/member1/deployment/member1-deployment.md) · [Раздел Member 1](../members/member1/README.md)
