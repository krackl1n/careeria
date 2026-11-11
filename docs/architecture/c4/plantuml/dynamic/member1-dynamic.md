# Динамическая диаграмма: загрузка резюме кандидата и привязка к профилю

Динамическая диаграмма C4 (**Dynamic Diagram**) фиксирует хронологическую последовательность взаимодействия компонентов и контейнеров платформы **Careeria** во времени в рамках ключевого рантайм-сценария домена **Member 1** (**Кандидаты и вакансии**):
> *«Инициализация загрузки резюме соискателем (`RequestResumeUpload`), ReBAC-авторизация прав владельца профиля в `OpenFGA`, выпуск криптографически подписанной ссылки `Presigned PUT URL` через `File Service`, фиксация версии резюме в статусе `UPLOADING`, прямая выгрузка бинарного файла в объектное хранилище `MinIO (S3)`, подтверждение загрузки (`ConfirmResumeUpload`) со сверкой и привязкой файла в `File Service`, атомарный перевод версии резюме в статус `READY` в единой транзакции с таблицей `outbox`, возврат синхронного ответа пользователю (`200 OK`) и гарантированная публикация события `CandidateResumeUploadedEvent` через CDC `Debezium` / `Kafka` с синхронизацией ReBAC-прав в `OpenFGA` и отправкой уведомления через `Notification Service`»*.

```plantuml
@startuml
!include architecture/c4/plantuml/dynamic/member1-dynamic.puml
@enduml
```

---

## 1. Назначение и сценарий взаимодействия

Загрузка резюме — центральный бизнес-процесс платформы для соискателя. В архитектуре `Candidate Service` (согласно декомпозиции уровня C4 L3) управление анкетой (`Candidate Profile Service`) и управление файловыми версиями резюме (`Resume Service`) разделены на два независимых агрегата, а сам жизненный цикл версии файла (`ResumeVersion`) проходит строгие состояния конечного автомата: `UPLOADING` $\rightarrow$ `CHECKING` $\rightarrow$ `READY` (либо `REJECTED` / `EXPIRED`).

Ключевые архитектурные паттерны, реализованные в данном сценарии:
- **Presigned URL Pattern (Direct-to-S3 Offloading)**: бинарный файл резюме (PDF/DOCX) загружается браузером напрямую в объектное хранилище `MinIO`, минуя `BFF`, `API Gateway`, `Candidate Service` и `File Service` — это устраняет буферизацию мультимегабайтных файлов в оперативной памяти микросервисов.
- **Двухфазная верификация и привязка файла (`Verify & Bind`)**: `Candidate Service` не доверяет слепо клиентскому запросу подтверждения, а синхронно сверяет факт загрузки, контрольную сумму (`ETag`) и результат проверки файла через `File Service` перед переводом резюме в статус `READY`.
- **Transactional Outbox + CDC (Debezium)**: перевод версии резюме в статус `READY`, обновление предпочтений основного резюме и запись события `CandidateResumeUploadedEvent` в таблицу `outbox` выполняются в единой ACID-транзакции PostgreSQL, после чего коннектор `Debezium` вычитывает событие из журнала WAL (`pgoutput`) и публикует его в `Apache Kafka`.

---

## 2. Спецификация участников взаимодействия

| Участник | Тип | Роль в сценарии |
|---|---|---|
| **Кандидат** | Person | Выбирает файл резюме (PDF/DOCX) на устройстве, инициирует загрузку и получает подтверждение публикации в личном кабинете. |
| **Web Application** | Container (Next.js, React) | Отображает форму профиля, отправляет JSON-метаданные файла, выполняет прямую `HTTP PUT`-выгрузку в MinIO по presigned URL и вызывает подтверждение загрузки. |
| **BFF** | Container (Go, net/http, grpc-go) | Сессионный фасад: проверяет HttpOnly-cookie `__Host-Careeria-Session` и CSRF-токен, транслирует запросы веб-клиента во внутренние gRPC-вызовы к API Gateway. |
| **API Gateway** | Container (Go, grpc-gateway) | Маршрутизирует вызовы в gRPC, добавляет доверенный identity context (`x-user-id`, `x-request-id`, `x-idempotency-key`) для downstream-сервисов. |
| **Candidate Service** | Container (Go, gRPC, Protobuf, pgx, sqlc) | Владелец домена профилей и резюме (`Resume Service`, `Candidate Profile Service`, `Resume Cleanup Worker`): проверяет права в OpenFGA, координирует генерацию ссылки и привязку файла через `File Service`, ведёт FSM версий резюме и пишет в `outbox`. |
| **OpenFGA** | Container (OpenFGA Engine) | Хранилище и движок вычисления ReBAC-графов доступа: проверяет право `can_upload_resume` и сохраняет новые кортежи отношений для созданной версии резюме. |
| **File Service** | Container (Go, gRPC, MinIO SDK) | Платформенный файловый сервис: валидирует MIME-тип и размер, генерирует временный presigned PUT URL, проверяет загруженный объект в MinIO и привязывает файл к резюме. |
| **Object Storage** | ContainerDb (MinIO / S3 API) | Приватное объектное хранилище бинарных файлов: принимает прямую загрузку от браузера по presigned URL, хранит резюме и аватары. |
| **Candidate DB** | ContainerDb (PostgreSQL 16) | Логическая база данных `candidate_db` в едином инстансе PostgreSQL: хранит таблицы `candidate_profiles`, `resumes`, `resume_versions`, `candidate_resume_preferences`, `resume_cleanup_jobs` и `outbox`. |
| **Debezium** | Container (Debezium PostgreSQL Connector) | Читает журнал транзакций WAL PostgreSQL (`pgoutput`), извлекает записи из таблицы `outbox` и публикует их в Kafka как бинарные Avro-события. |
| **Kafka** | ContainerQueue (Apache Kafka, KRaft) | Отказоустойчивая шина доменных событий. Топик `candidate.events` с `replication.factor=3` и `acks=all`. |
| **AuthZ Synchronization** | Container (Go, Kafka Consumer) | Платформенный воркер синхронизации авторизации: вычитывает доменные события из Kafka и атомарно обновляет ReBAC-кортежи доступа в `OpenFGA`. |
| **Notification Service** | Container (Go, Kafka Consumer) | Оркестратор мультиканальных оповещений: вычитывает событие из `candidate.events` с проверкой идемпотентности (`inbox`) и формирует уведомление кандидату. |
| **Notification Providers** | System_Ext (SendGrid / FCM / Twilio) | Внешние облачные шлюзы: доставляют Email/Push-подтверждение публикации резюме соискателю. |

---

## 3. Хронология выполнения этапов (Шаги 1–29)

### 3.1. Фаза А. Инициализация загрузки и получение Presigned URL (Шаги 1–12)
* **Шаг 1 (HTTPS / TLS 1.3, порт 443):** Кандидат выбирает файл резюме (`PDF` / `DOCX`, до 15 МБ) в форме личного кабинета `Web Application`. Клиент проверяет расширение и размер файла и формирует легковесный JSON-запрос с метаданными (`fileName`, `mimeType`, `sizeBytes`).
* **Шаг 2 (HTTPS / JSON, порт 443):** `Web Application` отправляет запрос `POST /api/candidate/resumes/upload-url` на сессионный фасад `BFF`, который валидирует cookie `__Host-Careeria-Session` и заголовок `X-CSRF-Token`.
* **Шаг 3 (gRPC / HTTP2, порт 50051):** `BFF` вызывает метод шлюза `RouteCandidateCall`, передавая доверенный контекст сессии пользователя.
* **Шаг 4 (gRPC / HTTP2, порт 50052):** `API Gateway` вызывает RPC-метод `CandidateService/RequestResumeUpload`. Внутри контейнера `gRPC Server` передаёт управление компоненту `Resume Service`.
* **Шаг 5 (gRPC / mTLS, порт 8081):** `Candidate Service` через `Authorization Client` обращается к `OpenFGA` для проверки ReBAC-отношения `(user:<UUID>, relation: "can_upload_resume", object: "candidate_profile:<UUID>")`.
* **Шаг 6 (gRPC / HTTP2, порт 50055):** После успешной авторизации `Resume Service` через `File Service Client` вызывает метод `FileService/BeginUpload(fileName, mimeType, sizeBytes)` у `File Service`.
* **Шаг 7 (S3 API / HTTPS, порт 9000):** `File Service` проверяет допустимость MIME-типа и квоту размера, обращается к `Object Storage (MinIO)` и формирует временный подписанный `Presigned PUT URL` (подпись `AWS Signature V4`, TTL 15 минут).
* **Шаг 8 (gRPC response):** `File Service` возвращает структуру `{presignedUrl, fileId, expiresAt}` в `Candidate Service`.
* **Шаг 9 (SQL / TCP, порт 5432):** `Resume Service` через `Resume Repo` сохраняет в `Candidate DB` (`candidate_db`) новую запись версии резюме `ResumeVersion` в начальном статусе **`UPLOADING`** и регистрирует задачу контроля истечения срока сессии в таблице `resume_cleanup_jobs`.
* **Шаги 10–12 (gRPC response → JSON / HTTPS):** `Candidate Service` возвращает `{presignedUrl, fileId, resumeVersionId}` через `API Gateway` и `BFF` в браузер со статусом `200 OK`.

### 3.2. Фаза Б. Прямая выгрузка файла и синхронное подтверждение загрузки (Шаги 13–23)
* **Шаг 13 (HTTPS / PUT, S3 API, порт 9000):** Браузер выполняет прямой запрос **`HTTP PUT`** по полученному `presignedUrl` непосредственно в `Object Storage (MinIO)`. Бинарный поток файла передаётся напрямую в S3, не нагружая память и сеть Go-сервисов. MinIO сохраняет объект и возвращает `200 OK` с заголовком `ETag`.
* **Шаги 14–16 (HTTPS → gRPC, порты 443 → 50051 → 50052):** Браузер отправляет подтверждение `POST /api/candidate/resumes/confirm` с `{resumeVersionId, fileId, etag}` через `BFF` и `API Gateway` в метод `CandidateService/ConfirmResumeUpload`.
* **Шаги 17–18 (gRPC / HTTP2, порт 50055 → S3 API, порт 9000):** Компонент `Resume Service` через `File Service Client` вызывает `FileService/VerifyAndBindFile(fileId, resumeVersionId)` (соответствует связи *«Проверяет и связывает файл»* на диаграмме L3). `File Service` проверяет в `MinIO` фактическое наличие объекта, совпадение размера/`ETag` и статус антивирусной проверки, после чего фиксирует привязку (`binding`) файла к резюме.
* **Шаг 19 (SQL Transaction / TCP, порт 5432):** В рамках единой физической транзакции PostgreSQL `BEGIN ... COMMIT` в базе `Candidate DB` (`candidate_db`):
  1. Компонент `Resume Repo` переводит версию резюме `ResumeVersion` в статус **`READY`** и снимает задачу аварийной очистки из `resume_cleanup_jobs`.
  2. При установке флага основного резюме обновляется запись в `candidate_resume_preferences` (с инкрементом `primary_selection_version`) и выставляется признак `has_resume = true` в `candidate_profiles`.
  3. Через компонент `Transactional Outbox` в таблицу `outbox` записывается событие `CandidateResumeUploadedEvent`.
* **Шаги 20–23 (gRPC response → UI Render):** `Candidate Service` возвращает `ConfirmResumeUploadResponse` (`200 OK`) с актуальным DTO резюме через `API Gateway` и `BFF` в браузер, где UI отображает готовую версию резюме.

### 3.3. Фаза В. Асинхронная доставка событий через CDC и реакция подписчиков (Шаги 24–29)
* **Шаг 24 (PostgreSQL WAL, порт 5432):** `Debezium Connector` без периодического опроса считывает новую строку таблицы `outbox` из логического слота репликации `careeria_candidate_slot` (`pgoutput`).
* **Шаг 25 (Kafka Protocol / TCP, порт 9092):** `Debezium` сериализует событие `CandidateResumeUploadedEvent` в бинарный формат Apache Avro и публикует в топик `candidate.events` кластера `Apache Kafka` (`acks=all`).
* **Шаги 26–27 (Kafka → OpenFGA API, порт 8081):** Воркер `AuthZ Synchronization` вычитывает событие из `candidate.events` и записывает в `OpenFGA` ReBAC-кортежи принадлежности новой версии резюме профилю кандидата (`candidate_profile:<UUID>#owner@user:<UUID>`), чтобы правила видимости корректно работали при откликах на вакансии.
* **Шаги 28–29 (Kafka → HTTPS / REST API, порт 443):** Микросервис `Notification Service` вычитывает событие в группе `careeria.notifications.group` с проверкой идемпотентности по таблице `inbox` и обращается к внешним шлюзам `Notification Providers` (`SendGrid` / `FCM`) для отправки кандидату подтверждения об успешной публикации резюме.

---

## 4. Сводная спецификация параметров динамического процесса

| № | Направление | Инициатор | Получатель | Операция / интерфейс | Протокол / Порт | Формат данных | Тип вызова | SLA / Таймаут |
|:---:|---|---|---|---|---|---|---|---|
| 1 | Запрос | Кандидат | Web Application | `requestResumeUploadForm()` | HTTPS / 443 | JSON DTO | Синхронный | < 50 мс |
| 2 | Запрос | Web Application | BFF | `POST /api/candidate/resumes/upload-url` | HTTPS / 443 | JSON + Cookie | Синхронный | < 100 мс |
| 3 | Запрос | BFF | API Gateway | `GatewayService/RouteCandidateCall` | gRPC / 50051 | Protobuf + Metadata | Синхронный | < 120 мс |
| 4 | Запрос | API Gateway | Candidate Service | `CandidateService/RequestResumeUpload` | gRPC / 50052 | Protobuf DTO | Синхронный | < 150 мс |
| 5 | Запрос | Candidate Service | OpenFGA | `OpenFGAService/Check(can_upload_resume)` | gRPC / 8081 | Protobuf ReBAC Tuple | Синхронный | < 20 мс |
| 6 | Запрос | Candidate Service | File Service | `FileService/BeginUpload` | gRPC / 50055 | Protobuf DTO | Синхронный | < 50 мс |
| 7 | Запрос | File Service | Object Storage | Генерация `Presigned PUT URL` (SigV4) | S3 API / 9000 | S3 Signed URL | Синхронный | < 30 мс |
| 8 | Ответ | File Service | Candidate Service | `BeginUploadResponse(presignedUrl, fileId)` | gRPC / 50055 | Protobuf Response | Синхронный | < 20 мс |
| 9 | Запрос | Candidate Service | Candidate DB | `INSERT resume_versions (UPLOADING)` | SQL / 5432 | SQL Statements | Синхронный | < 30 мс |
| 10 | Ответ | Candidate Service | API Gateway | `RequestResumeUploadResponse` | gRPC / 50052 | Protobuf Response | Синхронный | < 20 мс |
| 11 | Ответ | API Gateway | BFF | `gRPC Response forward` | gRPC / 50051 | Protobuf Response | Синхронный | < 20 мс |
| 12 | Ответ | BFF | Web Application | `HTTP 200 OK (presignedUrl, versionId)` | HTTPS / 443 | JSON Payload | Синхронный | < 40 мс |
| 13 | Выгрузка | Web Application | Object Storage | `PUT /careeria-resumes/{fileId}` | HTTPS / 9000 | Binary Stream (PDF/DOCX) | Прямой (Offload) | < 5.0 с |
| 14 | Запрос | Web Application | BFF | `POST /api/candidate/resumes/confirm` | HTTPS / 443 | JSON + Cookie | Синхронный | < 100 мс |
| 15 | Запрос | BFF | API Gateway | `GatewayService/RouteCandidateCall` | gRPC / 50051 | Protobuf + Metadata | Синхронный | < 120 мс |
| 16 | Запрос | API Gateway | Candidate Service | `CandidateService/ConfirmResumeUpload` | gRPC / 50052 | Protobuf DTO | Синхронный | < 200 мс |
| 17 | Запрос | Candidate Service | File Service | `FileService/VerifyAndBindFile` | gRPC / 50055 | Protobuf DTO | Синхронный | < 60 мс |
| 18 | Запрос | File Service | Object Storage | `StatObject / VerifyChecksum` | S3 API / 9000 | S3 Object Metadata | Синхронный | < 30 мс |
| 19 | Запрос | Candidate Service | Candidate DB | `SQL Transaction (READY + Outbox)` | SQL / 5432 | SQL Statements | Синхронный | < 50 мс |
| 20 | Ответ | Candidate Service | API Gateway | `ConfirmResumeUploadResponse` | gRPC / 50052 | Protobuf Response | Синхронный | < 20 мс |
| 21 | Ответ | API Gateway | BFF | `gRPC Response forward` | gRPC / 50051 | Protobuf Response | Синхронный | < 20 мс |
| 22 | Ответ | BFF | Web Application | `HTTP 200 OK (JSON Resume DTO)` | HTTPS / 443 | JSON Payload | Синхронный | < 40 мс |
| 23 | Ответ | Web Application | Кандидат | Рендеринг UI (активное резюме) | UI Event | DOM / React State | Клиентский | < 16 мс |
| 24 | Событие | Candidate DB | Debezium Connector | Потоковое чтение PostgreSQL WAL | WAL / 5432 | Бинарный `pgoutput` WAL | Асинхронный (CDC) | < 100 мс |
| 25 | Событие | Debezium Connector | Apache Kafka | `ProduceRecord(candidate.events)` | Kafka / 9092 | Apache Avro Binary | Асинхронный | < 50 мс |
| 26 | Событие | Apache Kafka | AuthZ Synchronization | `FetchRecord(candidate.events)` | Kafka / 9092 | Apache Avro Binary | Асинхронный | < 50 мс |
| 27 | Запрос | AuthZ Synchronization | OpenFGA | `WriteTuples(resume_version)` | gRPC / 8081 | Protobuf ReBAC Tuple | Асинхронный | < 30 мс |
| 28 | Событие | Apache Kafka | Notification Service | `FetchRecord(candidate.events)` | Kafka / 9092 | Apache Avro Binary | Асинхронный | < 50 мс |
| 29 | Событие | Notification Service | Notification Providers | `SendTransactionalEmail / Push` | HTTPS / 443 | JSON Payload | Асинхронный | < 1.5 с |

---

## 5. Матрица обработки граничных ситуаций и исключений

| Исключительная ситуация | Слой обнаружения | Доменная ошибка | Код ответа | Архитектурная реакция и компенсация |
|---|---|---|---|---|
| Пользователь не является владельцем профиля или лишён прав на загрузку резюме | OpenFGA / Authz Client | `ErrUnauthorizedResumeUpload` | `403 Forbidden` / `codes.PermissionDenied` | Прерывание операции до обращения к `File Service` и БД. Логирование попытки несанкционированного доступа в журнал аудита. |
| Недопустимый MIME-тип файла (не PDF/DOCX) или превышен лимит размера ($> 15$ МБ) | Resume Service / File Service | `ErrUnsupportedMimeType` / `ErrFileSizeExceeded` | `422 Unprocessable Entity` / `codes.InvalidArgument` | Отказ в генерации `Presigned PUT URL`. Возврат клиенту детализированной ошибки валидации `google.rpc.BadRequest`. |
| Обрыв сети при прямой выгрузке файла в MinIO (шаг 13 не завершён, шаг 14 не поступил) | Resume Cleanup Worker (Background) | `ErrUploadSessionExpired` | Перевод версии в `EXPIRED` | По истечении TTL (15 мин) фоновый `Resume Cleanup Worker` захватывает зависшую запись `UPLOADING` через `FOR UPDATE SKIP LOCKED`, сверяет статус с `File Service`, переводит `ResumeVersion` в `EXPIRED` и освобождает объект в MinIO. |
| Подтверждение загрузки (`ConfirmResumeUpload`) для файла, который не был загружен в MinIO или не прошёл антивирус | Candidate Service → File Service (шаги 17–18) | `ErrFileNotUploaded` / `ErrFileInfected` | `409 Conflict` / `codes.FailedPrecondition` | `File Service` отклоняет привязку файла, `ResumeVersion` переводится в статус `REJECTED`, небезопасный объект удаляется из S3. |
| Конфликт параллельного выбора основного резюме или обновления профиля (`version mismatch`) | Infrastructure (`pgx`/`sqlc`, шаг 19) | `ErrOptimisticLockConflict` | `409 Conflict` / `codes.Aborted` | Откат SQL-транзакции (`ROLLBACK`) при несовпадении `primary_selection_version`. Клиенту предлагается обновить данные профиля и повторить операцию. |
| Сетевой сбой доставки письма через внешний шлюз SendGrid / FCM | Notification Service (шаг 29) | `ErrExternalGatewayTimeout` | Ошибка консьюмера | Offset в Kafka не коммитится, событие перенаправляется в очередь повторной доставки (`Retry Topic` → `DLQ`) с защитой от дублей через таблицу `inbox`. |

---

[Ландшафт L0](../members/member1/landscape/member1-l0-landscape.md) · [Контекст L1](../members/member1/context%28L1%29/index.md) · [Контейнеры L2](../members/member1/context%28L1%29/containers%28L2%29/index.md) · [Развёртывание](../members/member1/deployment/member1-deployment.md) · [Раздел Member 1](../members/member1/README.md)