# Динамическая диаграмма: загрузка резюме кандидата и привязка к профилю

Динамическая диаграмма C4 (**Dynamic Diagram**) описывает сквозной сценарий взаимодействия контейнеров платформы при выполнении ключевого бизнес-процесса домена **Member 1**: загрузка резюме кандидатом, его привязка к профилю и последующая публикация доменного события через событийную платформу. Диаграмма фиксирует как **синхронную** цепочку вызовов (пользователь → UI → BFF → Gateway → сервисы → ответ), так и **асинхронную** (CDC через Debezium → Kafka → потребители событий).

```plantuml
@startuml
!include architecture/c4/plantuml/dynamic/member1-dynamic.puml
@enduml
```

---

## 1. Назначение и сценарий взаимодействия

Загрузка резюме — центральный бизнес-процесс платформы для кандидата. Профиль без резюме существенно снижает шансы на отклик от рекрутера, поэтому надёжность и производительность этого сценария критична для UX.

Ключевые архитектурные паттерны, реализованные в данном сценарии:
- **Presigned URL Pattern**: файл загружается браузером напрямую в объектное хранилище (MinIO), минуя API-уровень — снижает нагрузку на сервисы и устраняет буферизацию бинарных данных в памяти.
- **Transactional Outbox Pattern**: метаданные резюме и событие для Kafka записываются в единой SQL-транзакции — гарантирует консистентность без двухфазного коммита.
- **Change Data Capture (CDC)**: Debezium вычитывает события из outbox-таблицы через PostgreSQL WAL, публикует их в Kafka строго «ровно один раз» (exactly-once семантика).

---

## 2. Спецификация участников взаимодействия

| Участник | Тип | Роль в сценарии |
|---|---|---|
| **Кандидат** | Person | Инициирует загрузку резюме через UI платформы. |
| **Web Application** | Container (Next.js, React) | Отображает форму профиля, отправляет API-запросы, загружает файл по presigned URL и отображает результат. |
| **BFF** | Container (Go, grpc-go) | Сессионный фасад: проверяет HttpOnly-cookie, защищает от CSRF, транслирует запросы в gRPC к API Gateway. |
| **API Gateway** | Container (Go, grpc-gateway) | Маршрутизирует REST-вызовы в gRPC, добавляет доверенный identity context (userId, role) для downstream-сервисов. |
| **Candidate Service** | Container (Go, gRPC, pgx) | Владелец домена профилей: координирует проверку прав, запрашивает presigned URL у File Service, подтверждает загрузку и пишет Outbox. |
| **OpenFGA** | Container (OpenFGA Engine) | Вычисляет ReBAC-граф доступа: проверяет, имеет ли данный пользователь право `can_upload_resume` на объект профиля. |
| **File Service** | Container (Go, MinIO SDK) | Генерирует presigned PUT URL, взаимодействует с MinIO по S3 API, управляет lifecycle файлов и метаданными. |
| **Object Storage** | ContainerDb (MinIO / S3 API) | Хранилище бинарных файлов: принимает прямую загрузку от браузера по presigned URL, хранит резюме и аватары. |
| **Candidate DB** | ContainerDb (PostgreSQL 16) | Хранит профиль кандидата, метаданные версий резюме и таблицу `transactional_outbox` для CDC. |
| **Debezium** | Container (Debezium PostgreSQL Connector) | Читает WAL PostgreSQL, извлекает записи из outbox и публикует их в Kafka как Avro-события. |
| **Kafka** | ContainerQueue (Apache Kafka, KRaft) | Отказоустойчивая шина событий. Топик `candidate.events` с `replication.factor=3`. |
| **Vacancy Service** | Container (Go, gRPC, pgx) | Потребитель события `CandidateResumeUploadedEvent` — обновляет индекс релевантности кандидатов для вакансий. |
| **Notification Providers** | System_Ext (SendGrid / FCM / Twilio) | Доставляет Email/Push-подтверждение публикации резюме кандидату. |

---

## 3. Описание шагов сценария

### 3.1. Синхронная цепочка: запрос и генерация presigned URL (шаги 1–11)

| Шаг | Участник (источник) | Участник (цель) | Описание |
|:---:|---|---|---|
| 1 | Кандидат | Web Application | Кандидат заполняет форму профиля в браузере и инициирует загрузку файла резюме (PDF/DOCX). UI формирует multipart-запрос. |
| 2 | Web Application | BFF | Web-клиент передаёт запрос `RequestResumeUpload` через HTTPS с сессионной HttpOnly-cookie. BFF валидирует cookie и добавляет CSRF-токен. |
| 3 | BFF | API Gateway | BFF направляет авторизованный gRPC-запрос `RequestResumeUpload` через внутреннюю сеть кластера на порт 50051. |
| 4 | API Gateway | Candidate Service | API Gateway маршрутизирует RPC-вызов `RequestResumeUpload` к Candidate Service (порт 50052), добавляя доверенный identity context (userId). |
| 5 | Candidate Service | OpenFGA | Candidate Service проверяет полномочия: `check(user:<userId>, can_upload_resume, candidate-profile:<profileId>)`. При отказе возвращает gRPC-ошибку `PERMISSION_DENIED`. |
| 6 | Candidate Service | File Service | После успешной авторизации Candidate Service вызывает `GenerateUploadURL(fileName, mimeType, maxSizeBytes)` у File Service (порт 50055). |
| 7 | File Service | Object Storage | File Service обращается к MinIO по S3 API и генерирует presigned PUT URL с TTL 15 минут и подписью AWS Signature V4. |
| 8 | File Service | Candidate Service | File Service возвращает `{presignedUrl, fileId, expiresAt}` через gRPC. |
| 9–11 | Candidate Service → Gateway → BFF → Web | — | Presigned URL передаётся обратно по цепочке до браузера. Web-клиент получает `{presignedUrl, fileId}` в теле ответа. |

### 3.2. Прямая загрузка файла (шаг 12)

Браузер выполняет **HTTP PUT-запрос напрямую к MinIO** по полученному presigned URL. Файл резюме передаётся минуя все слои API платформы — это ключевое архитектурное решение, устраняющее:
- Двойную передачу бинарных данных (клиент → API → хранилище);
- Буферизацию файлов в памяти BFF/Gateway;
- Дополнительную задержку и нагрузку на вертикально масштабируемые ресурсы.

Успешная загрузка подтверждается HTTP-ответом `200 OK` от MinIO с `ETag` загруженного объекта.

### 3.3. Синхронная цепочка: подтверждение загрузки (шаги 13–20)

| Шаг | Участник (источник) | Участник (цель) | Описание |
|:---:|---|---|---|
| 13 | Web Application | BFF | Браузер отправляет `POST /api/candidate/resume/confirm` с `{fileId, etag}` — подтверждение успешной загрузки. |
| 14–15 | BFF → Gateway | Candidate Service | `ConfirmResumeUpload(fileId, etag)` RPC-вызов доставляется к Candidate Service. |
| 16 | Candidate Service | Candidate DB | В единой SQL-транзакции: (а) сохраняются метаданные резюме (`file_id`, `file_name`, `size`, `mime_type`, `version`); (б) обновляется статус профиля (`has_resume = true`); (в) вставляется запись в `transactional_outbox` (`CandidateResumeUploadedEvent`). |
| 17–19 | Candidate Service → Gateway → BFF | Web | Возвращается `200 OK` с актуальным DTO профиля кандидата (включая metadata резюме). |
| 20 | Web Application | Кандидат | UI обновляет страницу профиля: отображает иконку загруженного резюме, имя файла, дату загрузки и возможность скачивания. |

### 3.4. Асинхронная цепочка: CDC и событийная платформа (шаги 21–24)

| Шаг | Участник (источник) | Участник (цель) | Описание |
|:---:|---|---|---|
| 21 | Candidate DB | Debezium | Debezium PostgreSQL Connector непрерывно читает WAL-журнал через логический слот репликации `careeria_candidate_slot`. Обнаруживает INSERT в `transactional_outbox`. |
| 22 | Debezium | Kafka | Debezium сериализует событие `CandidateResumeUploadedEvent` в формат Avro (схема в Schema Registry) и публикует в топик `candidate.events` с `acks=all`. |
| 23 | Kafka | Vacancy Service | Vacancy Service, подписанный на `candidate.events`, получает событие и инициирует обновление индекса соответствия кандидата открытым вакансиям (matching engine). |
| 24 | Kafka | Notification Service | Notification Service получает событие, формирует Email/Push-уведомление для кандидата («Ваше резюме успешно опубликовано») и передаёт его внешним провайдерам. |

---

## 4. Ключевые архитектурные решения

### 4.1. Presigned URL вместо проксирования файла через API
Загрузка через presigned URL — стандартный паттерн работы с объектными хранилищами (AWS S3, GCS, MinIO). Альтернатива — загрузка через API-сервер — создаёт hotspot в BFF/Gateway при пиковой нагрузке (многие пользователи загружают CV одновременно). Presigned URL полностью устраняет эту проблему.

### 4.2. Transactional Outbox вместо dual-write
Запись метаданных файла и события для Kafka в **одной SQL-транзакции** гарантирует: либо оба записаны, либо ни один. Это исключает несогласованность, возникающую при сбое между двумя независимыми операциями (dual-write проблема). Outbox-запись читается Debezium через WAL без polling-запросов к СУБД.

### 4.3. CDC через Debezium вместо прямой записи в Kafka
Debezium читает WAL PostgreSQL, который уже фиксирует каждую транзакцию — это даёт **exactly-once** гарантию доставки события и устраняет необходимость в двухфазном коммите между СУБД и брокером.

---

[Ландшафт L0](../members/member1/landscape/member1-l0-landscape.md) · [Контекст L1](../members/member1/context%28L1%29/index.md) · [Контейнеры L2](../members/member1/context%28L1%29/containers%28L2%29/index.md) · [Развёртывание](../members/member1/deployment/member1-deployment.md) · [Раздел Member 1](../members/member1/README.md)

