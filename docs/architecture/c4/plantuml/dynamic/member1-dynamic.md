# Динамическая диаграмма: загрузка резюме кандидата и привязка к профилю

Динамическая диаграмма C4 (**Dynamic Diagram**) фиксирует хронологическую последовательность взаимодействия компонентов и контейнеров платформы **Careeria** во времени в рамках критически важного пользовательского сценария домена **Member 1** (**Кандидаты и вакансии**):
> *«Инициализация загрузки резюме соискателем (`RequestResumeUpload`), ReBAC-авторизация прав владельца профиля в `OpenFGA`, генерация криптографически подписанной ссылки `Presigned PUT URL` через `File Service`, прямая выгрузка бинарного файла в объектное хранилище `MinIO (S3)`, подтверждение загрузки (`ConfirmResumeUpload`), атомарный перевод версии резюме в статус `READY` в единой транзакции с таблицей `outbox`, возврат синхронного ответа пользователю (`200 OK`) и гарантированная публикация события `CandidateResumeUploadedEvent` через CDC `Debezium` / `Kafka` с обновлением релевантности в `Vacancy Service` и отправкой уведомления через `Notification Service`»*.

```plantuml
@startuml
!include architecture/c4/plantuml/dynamic/member1-dynamic.puml
@enduml
```

---

## 1. Назначение и сценарий взаимодействия

Загрузка резюме — центральный бизнес-процесс платформы для соискателя. Актуальное резюме необходимо для откликов на вакансии и ранжирования кандидата в поисковой выдаче рекрутеров, поэтому надёжность и пропускная способность этого сценария критичны для UX.

Ключевые архитектурные паттерны, реализованные в данном сценарии:
- **Presigned URL Pattern**: бинарный файл резюме (PDF/DOCX) загружается браузером напрямую в объектное хранилище (MinIO), минуя API-уровень — это снижает нагрузку на `BFF`, `API Gateway` и `Candidate Service` и устраняет буферизацию бинарных данных в памяти.
- **Transactional Outbox Pattern**: метаданные версии резюме (`ResumeVersion` в статусе `READY`), статус профиля и событие для Kafka записываются в единой ACID-транзакции PostgreSQL — гарантирует консистентность без двухфазного коммита.
- **Change Data Capture (CDC)**: коннектор Debezium вычитывает события из таблицы `outbox` через журнал предзаписи PostgreSQL WAL (`pgoutput`) и публикует их в Apache Kafka без polling-нагрузки на СУБД.

---

## 2. Спецификация участников взаимодействия

| Участник | Тип | Роль в сценарии |
|---|---|---|
| **Кандидат** | Person | Инициирует загрузку резюме и подтверждает его привязку через UI личного кабинета. |
| **Web Application** | Container (Next.js, React) | Отображает форму профиля, передаёт метаданные файла, выполняет прямую HTTP PUT-выгрузку в MinIO по presigned URL и отображает результат. |
| **BFF** | Container (Go, net/http, grpc-go) | Сессионный фасад: проверяет HttpOnly-cookie `__Host-Careeria-Session`, защищает от CSRF, транслирует запросы в gRPC к API Gateway. |
| **API Gateway** | Container (Go, grpc-gateway) | Маршрутизирует вызовы в gRPC, добавляет доверенный identity context (`x-user-id`, `x-request-id`, `x-idempotency-key`) для downstream-сервисов. |
| **Candidate Service** | Container (Go, gRPC, Protobuf, pgx, sqlc) | Владелец домена профилей и резюме: проверяет права через OpenFGA, запрашивает presigned URL у File Service, подтверждает загрузку и пишет в Outbox. |
| **OpenFGA** | Container (OpenFGA, PostgreSQL) | Вычисляет ReBAC-граф доступа: проверяет, имеет ли пользователь право `can_upload_resume` на объект профиля кандидата. |
| **File Service** | Container (Go, gRPC, MinIO SDK) | Валидирует параметры файла (MIME-тип, размер), генерирует временный presigned PUT URL, взаимодействует с MinIO по S3 API. |
| **Object Storage** | ContainerDb (MinIO / S3 API) | Приватное объектное хранилище бинарных файлов: принимает прямую загрузку от браузера по presigned URL, хранит резюме и аватары. |
| **Candidate DB** | ContainerDb (PostgreSQL 16) | Логическая база данных `candidate_db` в едином инстансе PostgreSQL: хранит профили кандидатов, метаданные версий резюме, задачи очистки и таблицу `outbox`. |
| **Debezium** | Container (Debezium PostgreSQL Connector) | Читает журнал транзакций WAL PostgreSQL (`pgoutput`), извлекает записи из таблицы `outbox` и публикует их в Kafka как Avro-события. |
| **Kafka** | ContainerQueue (Apache Kafka, KRaft) | Отказоустойчивая шина доменных событий. Топик `candidate.events` с `replication.factor=3` и `acks=all`. |
| **Vacancy Service** | Container (Go, gRPC, Protobuf, pgx, sqlc) | Потребитель события `CandidateResumeUploadedEvent` — обновляет индекс соответствия компетенций кандидата открытым вакансиям. |
| **Notification Service** | Container (Go, Kafka Consumer) | Оркестратор мультиканальных оповещений: вычитывает событие из `candidate.events` с проверкой идемпотентности (`inbox`) и формирует уведомление. |
| **Notification Providers** | System_Ext (SendGrid / FCM / Twilio) | Внешние облачные шлюзы: доставляют Email/Push-подтверждение публикации резюме кандидату. |

---

## 3. Хронология выполнения этапов (Шаги 1–25)

### 3.1. Фаза А. Синхронный контур: запрос и генерация Presigned URL (Шаги 1–11)
* **Шаг 1 (HTTPS / TLS 1.3, порт 443):** Кандидат выбирает файл резюме (`PDF` / `DOCX`) в форме профиля `Web Application` и инициирует загрузку (`requestResumeUploadForm()`). Клиент проверяет допустимое расширение и размер файла.
* **Шаг 2 (HTTPS / JSON, порт 443):** `Web Application` передаёт запрос с метаданными файла (`fileName`, `mimeType`, `sizeBytes`) на сессионный фасад `BFF`, который валидирует cookie `__Host-Careeria-Session` и заголовок `X-CSRF-Token`.
* **Шаг 3 (gRPC / HTTP2, порт 50051):** `BFF` вызывает метод шлюза `RouteCandidateCall`, передавая доверенный контекст пользователя (`x-user-id`, `x-roles`).
* **Шаг 4 (gRPC / HTTP2, порт 50052):** `API Gateway` выполняет вызов RPC-метода `CandidateService/RequestResumeUpload` с валидацией Protobuf DTO. Внутри контейнера `gRPC Server` передаёт вызов прикладному компоненту `Resume Service`.
* **Шаг 5 (gRPC / mTLS, порт 8081):** `Candidate Service` через `Authorization Client` обращается к движку `OpenFGA` для проверки ReBAC-отношения `(user:<UUID>, relation: "can_upload_resume", object: "candidate_profile:<UUID>")`.
* **Шаг 6 (gRPC / HTTP2, порт 50055):** После успешной авторизации `Candidate Service` через `File Service Client` вызывает метод `FileService/BeginUpload(fileName, mimeType, maxSizeBytes)` у платформенного сервиса `File Service`.
* **Шаг 7 (S3 API / HTTPS, порт 9000):** `File Service` проверяет MIME-тип и лимит размера, обращается к `Object Storage (MinIO)` по S3 API и генерирует временный `Presigned PUT URL` с TTL 15 минут и криптографической подписью AWS Signature V4.
* **Шаги 8–11 (gRPC response → JSON / HTTPS):** `File Service` возвращает `{presignedUrl, fileId, expiresAt}` в `Candidate Service` (шаг 8), откуда ответ синхронно проходит через `API Gateway` (шаг 9) и `BFF` (шаг 10) в браузер клиента со статусом `200 OK` (шаг 11).

### 3.2. Фаза Б. Прямая выгрузка файла и синхронное подтверждение (Шаги 12–20)
* **Шаг 12 (HTTPS / PUT, S3 API, порт 9000):** Браузер выполняет **HTTP PUT-запрос напрямую к MinIO** по полученному `presignedUrl`. Бинарный поток файла резюме передаётся минуя `BFF`, `API Gateway` и `Candidate Service`. Хранилище MinIO сохраняет объект и возвращает `200 OK` с заголовком `ETag`.
* **Шаг 13 (HTTPS / JSON, порт 443):** Браузер отправляет запрос `POST /api/candidate/resume/confirm` с `{fileId, etag}` на `BFF` для подтверждения успешной загрузки.
* **Шаги 14–15 (gRPC / HTTP2, порты 50051 → 50052):** `BFF` пересылает запрос через `API Gateway`, который вызывает RPC-метод `CandidateService/ConfirmResumeUpload(fileId, etag)` у `Candidate Service`.
* **Шаг 16 (SQL / TCP, порт 5432):** В рамках единой физической транзакции PostgreSQL `BEGIN ... COMMIT` в логической БД `Candidate DB` (`candidate_db`):
  1. Компонент `Resume Service` через `Resume Repo` сохраняет метаданные версии резюме (`file_id`, `file_name`, `byte_size`, `mime_type`, `version_number`) в таблице `resume_versions` и переводит её статус в `READY`.
  2. При выборе основного резюме обновляется запись в `candidate_resume_preferences` (с контролем `primary_selection_version`) и выставляется признак `has_resume = true` в `candidate_profiles`.
  3. Через компонент `Transactional Outbox` в служебную таблицу `outbox` сохраняется событие `CandidateResumeUploadedEvent`.
* **Шаги 17–20 (gRPC response → UI Render):** Сервис возвращает `ConfirmResumeUploadResponse` (`200 OK`) с актуальным DTO профиля через `API Gateway` и `BFF` обратно в браузер клиента, где UI отображает прикреплённое резюме и превью метаданных.

### 3.3. Фаза В. Асинхронная доставка событий и внешняя интеграция (Шаги 21–25)
* **Шаг 21 (PostgreSQL WAL, порт 5432):** `Debezium Connector` без периодического опроса (zero polling) считывает запись таблицы `outbox` из слота логической репликации `careeria_candidate_slot` (`pgoutput`).
* **Шаг 22 (Kafka Protocol / TCP, порт 9092):** `Debezium` сериализует событие `CandidateResumeUploadedEvent` в бинарный формат Apache Avro (схема в Apicurio Schema Registry) и публикует в топик `candidate.events` брокера `Apache Kafka` (`acks=all`).
* **Шаг 23 (Kafka Protocol / TCP, порт 9092):** Микросервис `Vacancy Service` асинхронно вычитывает событие из `candidate.events` и обновляет индекс релевантности открытых вакансий под навыки кандидата.
* **Шаг 24 (Kafka Protocol / TCP, порт 9092):** Микросервис `Notification Service` вычитывает событие в консьюмер-группе `careeria.notifications.group` с контролем идемпотентности по таблице `inbox`.
* **Шаг 25 (HTTPS / REST API, порт 443):** `Notification Service` обращается к `Notification Providers` (`SendGrid` / `FCM`) для доставки письма и Push-уведомления кандидату об успешной публикации резюме.

---

## 4. Сводная спецификация параметров динамического процесса

| № | Направление | Инициатор | Получатель | Операция / интерфейс | Протокол / Порт | Формат данных | Тип вызова | SLA / Таймаут |
|:---:|---|---|---|---|---|---|---|---|
| 1 | Запрос | Кандидат | Web Application | `requestResumeUploadForm()` | HTTPS / 443 | JSON DTO | Синхронный | < 50 мс |
| 2 | Запрос | Web Application | BFF | `POST /api/candidate/resume/upload-url` | HTTPS / 443 | JSON + Cookie | Синхронный | < 100 мс |
| 3 | Запрос | BFF | API Gateway | `GatewayService/RouteCandidateCall` | gRPC / 50051 | Protobuf + Metadata | Синхронный | < 120 мс |
| 4 | Запрос | API Gateway | Candidate Service | `CandidateService/RequestResumeUpload` | gRPC / 50052 | Protobuf DTO | Синхронный | < 150 мс |
| 5 | Запрос | Candidate Service | OpenFGA | `OpenFGAService/Check` | gRPC / 8081 | Protobuf ReBAC Tuple | Синхронный | < 20 мс |
| 6 | Запрос | Candidate Service | File Service | `FileService/BeginUpload` | gRPC / 50055 | Protobuf DTO | Синхронный | < 50 мс |
| 7 | Запрос | File Service | Object Storage | Генерация `Presigned PUT URL` (SigV4) | S3 API / 9000 | S3 Signed URL | Синхронный | < 30 мс |
| 8 | Ответ | File Service | Candidate Service | `BeginUploadResponse` | gRPC / 50055 | Protobuf Response | Синхронный | < 20 мс |
| 9 | Ответ | Candidate Service | API Gateway | `RequestResumeUploadResponse` | gRPC / 50052 | Protobuf Response | Синхронный | < 20 мс |
| 10 | Ответ | API Gateway | BFF | `gRPC Response forward` | gRPC / 50051 | Protobuf Response | Синхронный | < 20 мс |
| 11 | Ответ | BFF | Web Application | `HTTP 200 OK (presignedUrl, fileId)` | HTTPS / 443 | JSON Payload | Синхронный | < 40 мс |
| 12 | Выгрузка | Web Application | Object Storage | `PUT /careeria-resumes/{fileId}` | HTTPS / 9000 | Binary Stream (PDF/DOCX) | Прямой (Offload) | < 5.0 с |
| 13 | Запрос | Web Application | BFF | `POST /api/candidate/resume/confirm` | HTTPS / 443 | JSON + Cookie | Синхронный | < 100 мс |
| 14 | Запрос | BFF | API Gateway | `GatewayService/RouteCandidateCall` | gRPC / 50051 | Protobuf + Metadata | Синхронный | < 120 мс |
| 15 | Запрос | API Gateway | Candidate Service | `CandidateService/ConfirmResumeUpload` | gRPC / 50052 | Protobuf DTO | Синхронный | < 150 мс |
| 16 | Запрос | Candidate Service | Candidate DB | `SQL Transaction (BEGIN ... COMMIT)` | SQL / 5432 | SQL Statements | Синхронный | < 50 мс |
| 17 | Ответ | Candidate Service | API Gateway | `ConfirmResumeUploadResponse` | gRPC / 50052 | Protobuf Response | Синхронный | < 30 мс |
| 18 | Ответ | API Gateway | BFF | `gRPC Response forward` | gRPC / 50051 | Protobuf Response | Синхронный | < 30 мс |
| 19 | Ответ | BFF | Web Application | `HTTP 200 OK (JSON Profile DTO)` | HTTPS / 443 | JSON Payload | Синхронный | < 50 мс |
| 20 | Ответ | Web Application | Кандидат | Рендеринг UI (карточка резюме) | UI Event | DOM / React State | Клиентский | < 16 мс |
| 21 | Событие | Candidate DB | Debezium Connector | Потоковое чтение PostgreSQL WAL | WAL / 5432 | Бинарный `pgoutput` WAL | Асинхронный (CDC) | < 100 мс |
| 22 | Событие | Debezium Connector | Apache Kafka | `ProduceRecord(candidate.events)` | Kafka / 9092 | Apache Avro Binary | Асинхронный | < 50 мс |
| 23 | Событие | Apache Kafka | Vacancy Service | `FetchRecord(candidate.events)` | Kafka / 9092 | Apache Avro Binary | Асинхронный | < 100 мс |
| 24 | Событие | Apache Kafka | Notification Service | `FetchRecord(candidate.events)` | Kafka / 9092 | Apache Avro Binary | Асинхронный | < 50 мс |
| 25 | Событие | Notification Service | Notification Providers | `SendTransactionalEmail / Push` | HTTPS / 443 | JSON Payload | Асинхронный | < 1.5 с |

---

## 5. Матрица обработки граничных ситуаций и исключений

| Исключительная ситуация | Слой обнаружения | Доменная ошибка | Код ответа | Архитектурная реакция и компенсация |
|---|---|---|---|---|
| Пользователь не является владельцем профиля или лишён прав на загрузку резюме | OpenFGA / Authz Client | `ErrUnauthorizedResumeUpload` | `403 Forbidden` / `codes.PermissionDenied` | Прерывание операции до обращения к `File Service` и БД. Логирование попытки несанкционированного доступа в журнал аудита. |
| Недопустимый MIME-тип файла (не PDF/DOCX) или превышен лимит размера ($> 15$ МБ) | Resume Service / File Service | `ErrUnsupportedMimeType` / `ErrFileSizeExceeded` | `422 Unprocessable Entity` / `codes.InvalidArgument` | Отказ в генерации `Presigned PUT URL`. Возврат клиенту детализированной ошибки валидации `google.rpc.BadRequest`. |
| Обрыв сети при прямой выгрузке файла в MinIO (шаг 12 не завершён, шаг 13 не поступил) | Resume Cleanup Worker (Background) | `ErrUploadSessionExpired` | Перевод версии в `EXPIRED` | По истечении TTL (15 мин) фоновый `Resume Cleanup Worker` захватывает зависшую запись через `FOR UPDATE SKIP LOCKED`, переводит `ResumeVersion` в `EXPIRED` и удаляет осиротевший объект через `File Service`. |
| Подтверждение загрузки (`ConfirmResumeUpload`) для файла, который не был загружен в MinIO или не прошёл антивирус | Candidate Service / File Service | `ErrFileNotUploaded` / `ErrFileInfected` | `409 Conflict` / `codes.FailedPrecondition` | Версия резюме переводится в статус `REJECTED`, транзакция привязки к профилю отклоняется, небезопасный объект удаляется из S3. |
| Конфликт параллельного выбора основного резюме или обновления профиля (`version mismatch`) | Infrastructure (`pgx`/`sqlc`) | `ErrOptimisticLockConflict` | `409 Conflict` / `codes.Aborted` | Откат SQL-транзакции (`ROLLBACK`). Клиенту предлагается обновить данные профиля и повторить операцию. |
| Сетевой сбой доставки письма через внешний шлюз SendGrid / FCM | Notification Service | `ErrExternalGatewayTimeout` | Ошибка консьюмера | Offset в Kafka не коммитится, событие перенаправляется в очередь повторной доставки (`Retry Topic` → `DLQ`) с защитой от дублей через таблицу `inbox`. |

---

[Ландшафт L0](../members/member1/landscape/member1-l0-landscape.md) · [Контекст L1](../members/member1/context%28L1%29/index.md) · [Контейнеры L2](../members/member1/context%28L1%29/containers%28L2%29/index.md) · [Развёртывание](../members/member1/deployment/member1-deployment.md) · [Раздел Member 1](../members/member1/README.md)