# Communication Service — компоненты L3

Communication Service владеет диалогами, составом участников, сообщениями, реакциями, отметками о прочтении (watermarks) и метаданными вложений в процессе взаимодействия участников найма. Содержимое файлов-вложений хранится в S3-совместимом Object Storage, остальные данные — в Communication DB (PostgreSQL). Доступ к файлам выполняется через общий File Service, доставка сообщений в реальном времени — через связку Realtime Publisher и Centrifugo (WebSocket), а внешние оповещения — через Notification Service.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/component/backend/communication-service.puml
@enduml
```

## Границы и статус проекта

Это предлагаемая целевая архитектура и каталог сценариев, не описание уже реализованных RPC. Названия команд ниже задают проектируемый контракт; protobuf, миграции и обработчики ещё предстоит реализовать. В отличие от общих чат-платформ, коммуникации в Careeria строго привязаны к бизнес-контексту рекрутинга: откликам, собеседованиям и корпоративным ролям.

В одном backend-контейнере выделены два бизнес-компонента:

- **Conversation Service** — управление контекстами диалогов (Direct, Application, Interview Room), составом участников, их ролями, правами на участие и жизненным циклом бесед.
- **Messaging Service** — самостоятельные агрегаты сообщений, отметок прочтения и реакций: приём и валидация текста, квоты вложений, ответы (threading), редактирование, мягкое удаление, фиксация прочитанного и эмоциональных откликов.

Это не два отдельных микросервиса. Оба используют общую Communication DB и один внешний gRPC-контракт CommunicationService. Не нужны отдельные application services под каждое действие или тип диалога. Жизненный цикл и инварианты закреплены в компактных доменных агрегатах и доменных политиках, а не в транспортных gRPC-хэндлерах или сырых SQL-запросах.

| Данные / действие | Владелец | Граница |
|---|---|---|
| Текст сообщений, реакции, отметки прочтения, состав участников диалога | Communication Service | Не хранит файлы, учётные данные или анкеты кандидатов. |
| Привязка диалога к контексту найма (application ID, vacancy ID, interview ID) | Communication Service | Хранит ID контекста, но не управляет переходами этапов воронки отбора. |
| Байты файлов, upload session, MIME-тип, размер, антивирус, S3 object key, download URL | File Service / Object Storage | Communication Service хранит только проверенный file ID и метаданные вложения. |
| Регистрация, аутентификация, JWT, смена паролей, блокировка пользователей | Identity Service | user ID и корпоративная роль берутся из доверенного identity context. |
| Статус отклика, воронка этапов, назначение собеседований, принятие оффера | Hiring Service | Завершение этапа может переводить диалог в Read-Only через доменное событие. |
| Поддержание WebSocket-соединений, каналы пользователей, presence | Realtime Publisher / Centrifugo | Communication Service фиксирует событие в outbox; Centrifugo транслирует его в браузер. |
| Отправка push-уведомлений, email и SMS при офлайн-статусе адресата | Notification Service | Асинхронно считывает событие MessageSent из Kafka и решает вопрос доставки. |
| Отношения доступа к диалогам и контекстным ресурсам | OpenFGA | ReBAC-проверки членства и разрешений на чтение/отправку сообщений. |

Внутренние компоненты и их связи объявлены в communication-service.puml. model/communication-elements.puml содержит имя контейнера и Communication DB. File Service, Centrifugo, Notification Service и OpenFGA переиспользуются из общих объявлений. Все находятся внутри контура Careeria; серый цвет обозначает готовую платформенную инфраструктуру. Kafka, Schema Registry и Debezium не раскрываются на этой L3 как прямые клиенты: сервис пишет данные в локальный transactional outbox.

## Элементы диаграммы

### Внутренние компоненты

| Элемент | Технология / слой | Ответственность | Зависимости |
|---|---|---|---|
| gRPC Server | Go / gRPC | Проверяет структуру запроса, извлекает identity context, request ID и ключ идемпотентности, транслирует доменные ошибки в gRPC codes. | Conversation Service, Messaging Service |
| Conversation Service | Go Application Service | Координирует создание диалогов, добавление/исключение участников, архивирование и блокировку бесед. Вызывает методы агрегата Conversation. | Authorization Client, Conversation Repo, Transactional Outbox |
| Messaging Service | Go Application Service | Координирует отправку сообщений, проверки через Messaging Policy, редактирование, удаление, реакции и отметки прочтения. | Messaging Policy, Authorization Client, Message Repo, Conversation Repo, File Service Client, Transactional Outbox |
| Messaging Policy | Go Domain Policy | Межагрегатная валидация: проверяет активность диалога, членство отправителя, лимиты текста и допустимые квоты вложений. | Отсутствуют (чистый Go) |
| Authorization Client | Go / OpenFGA SDK | Адаптирует ReBAC-проверки прав доступа к диалогам на основе контекста компании/вакансии. | OpenFGA |
| Conversation Repo | Go / pgx, sqlc | Загружает и сохраняет агрегаты Conversation и списки участников; выполняет выборки диалогов пользователя. | Communication DB |
| Message Repo | Go / pgx, sqlc | Сохраняет Message, MessageAttachment, ReadReceipt, Reaction; выполняет пагинацию курсором. | Communication DB |
| File Service Client | Go / gRPC | Проверяет статус и метаданные загруженных файлов, регистрирует постоянный binding вложения, запрашивает download URL. | File Service |
| Transactional Outbox | Go / pgx, sqlc | Сохраняет доменные события в таблице outbox в той же SQL-транзакции, что и изменение состояния. | Communication DB |
| Attachment Cleanup Worker | Go Background Worker | Отслеживает отклонённые или отвязанные вложения, запрашивает освобождение ресурсов у File Service. | Message Repo, File Service Client |

Repository и Outbox получают единый transaction handle (pgx.Tx) от прикладного слоя. Техническая изоляция транзакций вынесена в общую утилиту WithinTransaction.

### Соседние контейнеры

| Элемент | Технология | Назначение / связь |
|---|---|---|
| API Gateway | Go, grpc-gateway, grpc-go, OpenTelemetry | Маршрутизирует клиентские REST/HTTP-запросы и передаёт доверенный identity context. |
| Communication DB | PostgreSQL 16 | Хранение диалогов, сообщений, участников, реакций, watermarks, outbox и ключей идемпотентности. |
| OpenFGA | OpenFGA, PostgreSQL | Хранилище кортежей ReBAC-авторизации и проверка прав на диалоги. |
| File Service | Go, gRPC, MinIO Go SDK | Управление жизненным циклом бинарных объектов вложений. |
| Object Storage | MinIO, S3 API | Приватное хранилище медиафайлов и документов чата. |

## Доменная модель и хранение

Вместо монолитного агрегата «Чат со всеми сообщениями», приводящего к блокировкам БД, предметная область декомпозирована на компактные агрегаты:

| Сущность | Основные данные | Инварианты |
|---|---|---|
| Conversation | conversation ID, type (DIRECT / APPLICATION / INTERVIEW), context ID, status, version, timestamps | Контекстный диалог привязан к валидному context ID. В DIRECT строго 2 участника; повторное создание между ними идемпотентно возвращает существующий ID. |
| ConversationParticipant | conversation ID, user ID, role (CANDIDATE / RECRUITER / HIRING_MANAGER / OBSERVER), joinedAt, isMuted | Уникальность пары (conversation ID, user ID). В архивный или закрытый диалог нельзя добавлять участников. |
| Message | message ID, conversation ID, sender ID, replyToMessageId, content, status (SENT / EDITED / SOFT_DELETED), version, timestamps | Текст от 1 до 4000 символов (или непустые вложения). Редактирование разрешено только автору в течение 24 часов. Удалённое сообщение не редактируется. |
| MessageAttachment | attachment ID, message ID, file ID, fileName, sizeBytes, mimeType, createdAt | Не более 10 вложений на одно сообщение. Суммарный объём файлов в одном сообщении — не более 50 МБ. Файл должен иметь статус READY в File Service. |
| ReadReceipt | conversation ID, user ID, lastReadMessageId, lastReadAt | Монотонный сдвиг указателя прочтения вперёд по временной шкале; нельзя «откатить» прочитанное назад. |
| MessageReaction | message ID, user ID, emojiCode, createdAt | Уникальность тройки (message ID, user ID, emojiCode). Пользователь может повторным кликом только снять свою реакцию. |
| OutboxEvent | event ID, aggregate ID, aggregate type, event type, schema version, payload, occurredAt | Фиксируется атомарно в транзакции с изменением бизнес-сущности; обеспечивает гарантию At-Least-Once доставки. |
| IdempotencyKey | key, user ID, operation, response payload, lockedUntil, createdAt | Предотвращает дублирование сообщений при нестабильной мобильной сети и ретраях. |

Межагрегатная валидация изолирована в MessagingPolicy: отправка сообщения требует одновременно статуса Conversation.ACTIVE, наличия отправителя в списке активных ConversationParticipant и соблюдения лимитов на размер вложений.

Предлагаемые таблицы: conversations, conversation_participants, messages, message_attachments, conversation_read_receipts, message_reactions, idempotency_keys, communication_outbox.

Внутри Communication DB используются внешние ключи, каскадные правила и частичные индексы. Между сервисами связь осуществляется исключительно по идентификаторам: прямого доступа к профилям Identity DB или откликам Hiring DB нет.

В S3 отправляются исключительно бинарные файлы. Communication DB не хранит файлы, base64-строки или presigned URL. S3 key принадлежит File Service, Communication Service оперирует стабильным file ID.

## Статусы и жизненный цикл

| Ось | Предлагаемые значения | Правило |
|---|---|---|
| Жизненный цикл диалога | ACTIVE, READ_ONLY, ARCHIVED | ACTIVE разрешает отправку сообщений. READ_ONLY устанавливается автоматически при закрытии отклика или оффера (сообщения читать можно, писать нельзя). ARCHIVED скрывает диалог из активного списка. |
| Тип диалога | DIRECT, APPLICATION, INTERVIEW | Определяет политику наполнения участниками и контекстную привязку к бизнес-сущностям платформы. |
| Состояние сообщения | SENT, EDITED, SOFT_DELETED | Редактирование сохраняет отметку времени edited_at. При SOFT_DELETED текст заменяется на техническую заглушку, вложения отвязываются, но тред не разрушается. |
| Тип участия | CANDIDATE, RECRUITER, INTERVIEWER, SYSTEM_BOT | Определяет системные права на управление составом участников и доступ к приватным заметкам. |

Переходы состояний сообщений: SENT → EDITED (многократно в течение 24 часов) и SENT / EDITED → SOFT_DELETED. Восстановление удалённых сообщений не предусмотрено.

## Каталог сценариев

| № | Сценарий | Компонент | Результат |
|---|---|---|---|
| 1 | CreateDirectConversation | Conversation Service | Инициализирован или возвращён существующий диалог между 2 пользователями. |
| 2 | CreateContextConversation | Conversation Service | Создан групповой диалог по отклику или интервью с привязкой участников. |
| 3 | AddParticipantToConversation | Conversation Service | В диалог добавлен участник с проверкой ReBAC-прав инициатора. |
| 4 | RemoveParticipantFromConversation | Conversation Service | Участник исключён из диалога; новые сообщения ему недоступны. |
| 5 | ArchiveConversation | Conversation Service | Диалог переведён в статус ARCHIVED для инициатора. |
| 6 | CloseConversationToReadOnly | Conversation Service | Диалог переведён в READ_ONLY (по событию завершения найма). |
| 7 | GetConversation | Conversation Service | Получены метаданные диалога и список участников. |
| 8 | ListMyConversations | Conversation Service | Выборка диалогов текущего пользователя с пагинацией курсором и счетчиком непрочитанных. |
| 9 | SendMessage | Messaging Service | Сообщение проверено, сохранено, зарегистрировано в outbox для отправки в Centrifugo. |
| 10 | EditMessage | Messaging Service | Текст сообщения обновлен в пределах окна редактирования (24ч). |
| 11 | DeleteMessage | Messaging Service | Мягкое удаление сообщения с фиксацией события отзыва. |
| 12 | ListMessages | Messaging Service | Постраничная выборка сообщений диалога курсором по времени/ID. |
| 13 | MarkMessagesAsRead | Messaging Service | Сдвинут watermark прочтения, событие передано в outbox. |
| 14 | AddReaction | Messaging Service | Добавлена реакция на сообщение (эмодзи). |
| 15 | RemoveReaction | Messaging Service | Удалена ранее поставленная реакция. |
| 16 | GetAttachmentDownloadURL | Messaging Service | Запрошен временный presigned URL у File Service на скачивание вложения. |
| 17 | CleanupDetachedAttachments | Attachment Cleanup Worker | Запланировано освобождение непривязанных или удалённых файлов в File Service. |

### 1. CreateDirectConversation — создать личный диалог

**Инициатор и вход:** авторизованный пользователь, recipient_user_id, ключ идемпотентности.

**Проверки и выполнение:** Conversation Service проверяет, что инициатор и получатель — разные пользователи, запрашивает OpenFGA на отсутствие взаимных блокировок. В единой SQL-транзакции с помощью предикативного лока или детерминированного хэша пары (min(user1, user2), max(user1, user2)) проверяется наличие активного DIRECT-диалога. Если диалог уже существует, возвращается его ID без дублирования. Если нет — создаётся новый Conversation со статусом ACTIVE и два ConversationParticipant. Записывается событие ConversationCreated в outbox.

**Результат и повтор:** возвращается conversation_id. Повтор запроса с тем же idempotency key возвращает закешированный результат.

### 9. SendMessage — отправить сообщение

**Инициатор и вход:** участник диалога, conversation_id, текст сообщения, опциональный reply_to_message_id, список file_id проверенных вложений, ключ идемпотентности.

**Проверки и выполнение:** Messaging Service загружает агрегат Conversation и список участников. MessagingPolicy проверяет:

- Диалог находится в статусе ACTIVE (не READ_ONLY и не ARCHIVED).
- Инициатор является активным участником данного диалога.
- Валидность текста (1–4000 символов, либо наличие хотя бы одного валидного вложения).
- Суммарный лимит вложений не превышает 50 МБ (до 10 файлов).

Если переданы file_id, сервис через File Service Client подтверждает их статус (READY) и принадлежность инициатору, фиксируя binding.

В единой SQL-транзакции сохраняется запись messages, записи message_attachments, обновляется собственный conversation_read_receipts отправителя на ID нового сообщения, и формируется событие MessageSent в таблице communication_outbox.

**Результат и повтор:** возвращается message_id и таймстемп. При ретрае по ключу идемпотентности возвращается созданное сообщение без дублирования рассылки.

### 10. EditMessage — изменить сообщение

**Инициатор и вход:** автор сообщения, message_id, новый текст, expected_version.

**Проверки:** сервис проверяет, что отправитель является автором (sender_id == identity.user_id), сообщение не имеет статуса SOFT_DELETED, а с момента создания прошло не более 24 часов.

**Выполнение и результат:** текст сообщения обновляется, статус переходит в EDITED, инкрементируется version, устанавливается edited_at = NOW(). В той же транзакции создаётся outbox-событие MessageEdited. Если версия не совпала — возвращается VersionConflict.

### 11. DeleteMessage — мягкое удаление сообщения

**Инициатор и вход:** автор сообщения или администратор диалога, message_id.

**Проверки:** проверка авторства либо ReBAC-разрешения на модерацию в OpenFGA.

**Выполнение и результат:** поле content очищается (заменяется маркером «Сообщение удалено»), статус переводится в SOFT_DELETED, фиксируется deleted_at. Вложения помечаются отвязанными, и создаётся задача для Attachment Cleanup Worker на освобождение их постоянных bindings в File Service. В outbox пишется событие MessageDeleted.

### 13. MarkMessagesAsRead — отметить прочтение

**Инициатор и вход:** участник диалога, conversation_id, up_to_message_id.

**Проверки и выполнение:** проверяется участие в диалоге и существование целевого сообщения. В таблице conversation_read_receipts с помощью конструкции ON CONFLICT (conversation_id, user_id) DO UPDATE указатель смещается только при условии, что messages.created_at целевого сообщения строго больше текущего зафиксированного.

**Результат:** в outbox сохраняется событие MessagesRead для обновления бейджей непрочитанных у отправителей через Centrifugo.

### 16. GetAttachmentDownloadURL — получить ссылку на скачивание файла

**Инициатор и вход:** участник диалога, message_id, attachment_id.

**Проверки:** проверяется, что вызывающий состоит в диалоге, сообщение не удалено, а запрашиваемое вложение принадлежит данному сообщению.

**Выполнение:** Communication Service обращается к File Service по внутреннему gRPC-интерфейсу, передавая подтверждённый file_id. File Service генерирует краткоживущий presigned URL (TTL до 5 минут) на приватный S3 bucket. Ссылка передаётся клиенту; в базы данных и события она не сохраняется.

## Транзакции, безопасность и интеграция

### Авторизация внутри бизнес-сервисов

API Gateway берет на себя аутентификацию (проверку JWT/сессии) и пробрасывает доверенный контекст X-User-ID, X-Company-ID и X-User-Roles.

Внутри Communication Service:

- Проверка участия в личных и контекстных чатах выполняется по локальной таблице conversation_participants (быстрый путь без сетевых запросов).
- Проверка прав на создание корпоративных чатов, доступ к откликам и назначение интервью делегируется OpenFGA SDK на основе кортежей отношений user:<id>#member@company:<id>.

Ошибка авторизации возвращает PermissionDenied без раскрытия деталей о существовании скрытого диалога (NotFound).

### SQL и внешние операции

Запись сообщения, регистрация вложений, сдвиг watermark и вставка записи в outbox выполняются строго в одной PostgreSQL-транзакции.

Сетевые вызовы к File Service (получение presigned URL, валидация) выполняются до открытия или после фиксации SQL-транзакции. Запрещено удерживать транзакцию БД на время межсервисных вызовов.

### Предлагаемые события

| События | Изменение | Минимальный payload |
|---|---|---|
| ConversationCreated | Создан новый диалог | conversation ID, type, context ID, participant IDs, created_at |
| ConversationStatusChanged | Смена статуса (READ_ONLY/ARCHIVED) | conversation ID, previous/new status, reason |
| MessageSent | Отправлено сообщение | message ID, conversation ID, sender ID, recipient IDs, snippet/content, attachments_count, sent_at |
| MessageEdited | Отредактирован текст | message ID, conversation ID, new content, edited_at |
| MessageDeleted | Сообщение удалено | message ID, conversation ID, deleted_at |
| MessagesRead | Обновлена граница прочитанного | conversation ID, reader user ID, up_to_message_id, read_at |
| ReactionToggled | Добавлена/снята реакция | message ID, conversation ID, user ID, emojiCode, is_added |

События записываются в communication_outbox. Коннектор Debezium считывает WAL PostgreSQL и публикует типизированные Avro-сообщения в Kafka-топик careeria.communication.events.v1. Схемы валидируются через Apicurio Registry.

Потребители:

- **Realtime Publisher:** транслирует MessageSent, MessagesRead, ReactionToggled в Centrifugo для немедленного отображения в UI активных пользователей.
- **Notification Service:** обрабатывает MessageSent, проверяет presence-статус получателя и при необходимости формирует push/email уведомление.

### Ошибки и наблюдаемость

Коды ошибок контракта: Unauthenticated, PermissionDenied, NotFound, ValidationFailed, ConversationClosed, AttachmentQuotaExceeded, EditWindowExpired, Conflict.

Метрики: latency создания сообщений, задержка публикации из outbox (Debezium lag), объём загружаемых вложений, количество активных диалогов. Распределённая трассировка сквозная: traceparent и Request-ID передаются от API Gateway через gRPC context в SQL-комментарии и метаданные событий Kafka.

## Что не включено автоматически

| Вопрос | Решение первой версии / следующий контракт |
|---|---|
| Сквозное E2E-шифрование сообщений | В первой версии не реализуется. Сообщения шифруются на уровне БД (at-rest) и TLS (in-transit). E2E требует клиентского обмена ключами. |
| Полнотекстовый поиск по всей истории сообщений | В первой версии поиск ограничен фильтрацией по диалогу в PostgreSQL; выделенный Elasticsearch/Manticore кластер — отдельное расширение. |
| Голосовые и видеосообщения (кружочки, аудиозаметки) | В первой версии передаются как обычные файлы-вложения (аудио/видео MIME-типы) без потокового плеера и транскрибации речи. |
| Видеозвонки внутри чата | Реализуются через интеграцию со специализированным сервисом Hiring Service и медиасервером LiveKit. |
| Гарантированное удаление сообщений без следа (hard delete) | Исключено требованиями безопасности и аудита найма; поддерживается только soft delete с сохранением факта отправки. |

[Все L3](../README.md) · [Assessment Service](assessment-service.md) · [Hiring Service](hiring-service.md) · [Notification Service](notification-service.md)