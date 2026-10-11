# Integration Service — компоненты L3

Integration Service принимает вебхуки внешних систем и доставляет им исходящие вебхуки Careeria. Он владеет подключениями компаний, подписками, проверкой подлинности сообщений, нормализацией контрактов, идемпотентным приёмом и надёжной доставкой. Бизнес-состояние вакансии, кандидата или отклика меняет соответствующий доменный сервис, не сервис интеграций.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/component/backend/integration-service.puml
@enduml
```

## Границы и основные решения

Это целевая архитектура; RPC, HTTP endpoints, лимиты и payload ниже **предлагаемые**, не уже реализованные контракты.

В одном контейнере разделены три бизнес-компонента:

- **Integration Management Service** — подключения компаний, секреты, подписки, разрешённые типы событий, журнал и ручные повторы.
- **Inbound Webhook Service** — подлинность запроса, provider mapping, дедупликация и запись входящего события.
- **Outbound Webhook Service** — планирование доставок из внутренних событий и исполнение попыток по установленной политике.

HTTP и gRPC — разные входные адаптеры. HTTP webhook endpoint не требует пользовательской BFF-сессии: отправитель аутентифицируется отдельной подписью своего подключения. Управление настройками требует доверенной пользовательской identity и прав компании через OpenFGA.

| Направление | Цепочка | Где обеспечивается надёжность |
|---|---|---|
| Приём | Внешний отправитель → API Gateway → Webhook HTTP Handler → Inbound Webhook Service → Integration DB: inbox + outbox | Подтверждение только после локального SQL commit; повторы дедуплицируются. |
| Применение принятого изменения | Integration outbox → существующий Debezium → Kafka → Hiring / другой владелец ресурса | Доменный consumer самостоятельно проверяет полномочия источника, бизнес-правила, версию и повтор. |
| Планирование отправки | Kafka → Events Consumer → Outbound Webhook Service → Integration DB: processed event + deliveries | Kafka offset подтверждается после записи всех доставок данного события. |
| Доставка | Worker → Outbound Webhook Service → Webhook HTTP Client → внешний endpoint | Устойчивый delivery ID, lease, attempts, bounded retry и ручной redrive. |

Integration DB — собственное PostgreSQL-хранилище сервиса. Inbox, настройки, deliveries и outbox не размещаются в Hiring DB и не живут только в памяти или Kafka offsets. Сервис не делает прямой Kafka publish после SQL commit: входящие нормализованные события идут через уже принятую общую outbox/CDC-схему. Debezium не раскрывается на L3 как внутренний компонент; его путь показан на L2 и динамической диаграмме.

Контейнер, имя границы и БД переиспользуются из `model/integration-elements.puml`. Внутренние компоненты остаются в `component/backend/integration-service.puml`. Kafka, Schema Registry, OpenFGA и Integration DB находятся внутри Careeria и окрашены как готовая инфраструктура; Employer HR System — действительно внешняя система. Последняя представляет класс внешних отправителей/получателей, а не единственного жёстко зашитого поставщика.

## Элементы диаграммы

### Компоненты внутри Integration Service

| Элемент | Технология / слой | Обязанность | Зависимости |
|---|---|---|---|
| Webhook HTTP Handler | Go / net/http | Сохраняет исходные байты тела и подписанные headers, ограничивает размер/время чтения, возвращает receipt/status. Не применяет изменения найма. | Inbound Webhook Service |
| gRPC Server | Go / gRPC | Команды управления, просмотр журнала и redrive; форма DTO и преобразование ошибок. | Integration Management Service |
| Events Consumer | Go / Kafka Client | Читает разрешённые внутренние события, получает Avro schema и передаёт данные для планирования. Управляет offset после SQL commit. | Schema Registry, Outbound Webhook Service |
| Webhook Delivery Worker | Go Background Worker | Запускает ограниченные порции due deliveries; обеспечивает конкурентное исполнение через lease и прикладной сценарий. | Outbound Webhook Service |
| Inbound Webhook Service | Go Application Service | Проверяет подключение, подпись/свежесть, tenant и payload; нормализует и атомарно фиксирует inbox/outbox. | Inbound Webhook Adapter, Integration Repo, Transactional Outbox |
| Integration Management Service | Go Application Service | Управляет configuration lifecycle, ключами, подписками и разрешённым просмотром/повторами доставок. | Authorization Client, Integration Repo |
| Outbound Webhook Service | Go Application Service | Фильтрует события по компании и подпискам, сохраняет payload snapshots, захватывает jobs, подписывает и исполняет попытки. | Integration Repo, Outbound Webhook Adapter, Webhook HTTP Client |
| Inbound Webhook Adapter | Go / JSON, HMAC | Версионированные provider mappings, проверка подписи по сырым байтам и разбор проверенного payload. | Без прямых сетевых/SQL зависимостей |
| Outbound Webhook Adapter | Go / JSON, HMAC | Сериализация версионированного внешнего контракта и подпись доставки. Может переиспользовать криптографическую библиотеку с входным адаптером. | Без прямых сетевых/SQL зависимостей |
| Authorization Client | Go / OpenFGA SDK | Проверка полномочий просмотра/управления интеграциями компании. Не проверяет подпись стороннего webhook. | OpenFGA |
| Integration Repo | Go / pgx, sqlc | SQL-модуль: подключения, subscriptions, mappings, encrypted keys, inbox, processed events, deliveries, attempts, leases и журнал. Прикладные сервисы используют отдельные узкие repository ports, а не универсальный CRUD-интерфейс. | Integration DB |
| Webhook HTTP Client | Go / net/http | Исходящий HTTPS POST, destination policy, защищённое соединение, ограниченные timeout/response size; без redirects и скрытых POST retries. | Внешний webhook endpoint |
| Transactional Outbox | Go / pgx, sqlc | Нормализованное входящее событие в той же транзакции, что inbox и отметка дедупликации. | Integration DB |

Tx Manager — общая техническая обвязка транзакций, не отдельный бизнес-сервис на этой диаграмме. Репозитории и outbox получают одно transaction handle. Сетевой вызов не выполняется с удержанием SQL-lock до ответа внешнего сервера.

### Соседние контейнеры

| Элемент | Роль |
|---|---|
| API Gateway | Отдельный raw HTTP route для вебхуков; пользовательские команды управления идут через обычный REST-to-gRPC путь. |
| Kafka | Источник внутренних событий для исходящих доставок; нормализованные входящие сообщения появляются через outbox/CDC. |
| Schema Registry | Avro-схемы для Events Consumer; это не HTTP JSON schema registry внешнего provider. |
| Integration DB | Настройки, inbox, доставки, попытки, ключи идемпотентности и outbox. |
| OpenFGA | Управление интеграциями в контексте компании. |
| Employer HR System | Пример внешнего отправителя и получателя HTTPS webhooks. |

## Модель данных и инварианты

| Сущность | Основные поля | Правила |
|---|---|---|
| IntegrationConnection | ID, company ID, provider type, external tenant reference, lifecycle, inbound route token/ID, allowed inbound types, mapping version, version | Подключение принадлежит одной компании; company ID не принимается из внешнего payload как источник доверия. Route ID не заменяет signing secret. |
| WebhookSubscription | ID, connection ID, endpoint revision, destination host/path, event allowlist, payload schema version, lifecycle, version | Одно назначение и согласованный внешний контракт. Подписка не даёт доступа к событиям других компаний. |
| SigningKey | ID, connection/subscription ID, direction, key version, encrypted secret, encryption key ID, active/retiring/revoked, deadlines | Разные ключи для входа и выхода, отдельных получателей и компаний. Secret не возвращается в обычном GET. |
| InboundReceipt | receipt ID, connection ID, provider message ID, digest исходного тела, нормализованный event ID, schema/mapping version, receivedAt, status | Уникальная пара connection ID + provider message ID. Повтор с другим digest не является новым корректным сообщением. |
| ProcessedEvent | consumer name, source event ID, результат/причина игнорирования | Устойчивая дедупликация Kafka до commit offset. Нельзя положиться только на in-memory set. |
| WebhookDelivery | delivery ID, subscription ID/revision, source event ID, company ID, неизменяемые payload bytes/digest, status, attemptCount, nextAttemptAt, lease owner/token/until, delivery deadline | Уникальность subscription ID + source event ID. Один delivery ID на все повторы; payload и destination snapshot не меняются тихо. |
| DeliveryAttempt | attempt ID, delivery ID, ordinal, lease token, signing key version, startedAt/finishedAt, HTTP status, error class, bounded response metadata | Каждая попытка отдельна. Secrets, подписи и полное содержимое ответа не попадают в журнал. |
| OutboxEvent | ID, aggregate ID/version, event type/schema version, occurredAt, минимальный payload | Inbox и outbox фиксируются одной SQL-транзакцией. |

Предлагаемые таблицы: `integration_connections`, `webhook_subscriptions`, `signing_keys`, `inbound_receipts`, `processed_events`, `webhook_deliveries`, `webhook_delivery_attempts`, `idempotency_keys`, `outbox`. Config lifecycle и retry FSM реализуются доменными методами Connection/Subscription/Delivery, не установкой произвольного `status` из HTTP handler.

### Состояния

| Объект | Состояния | Семантика |
|---|---|---|
| Подключение | ACTIVE, PAUSED, REVOKED | PAUSED блокирует новый приём и планирование/отправку; REVOKED терминален и отзывает credentials. |
| Подписка | PENDING_VERIFICATION, ACTIVE, PAUSED, DELETED | Новый endpoint сначала подтверждается тестовым handshake; DELETED не означает стирание журнала. |
| Входящий receipt | ACCEPTED, IGNORED | ACCEPTED — inbox/outbox commit, не результат применения доменом. IGNORED — подписанное, известное, но явно не обрабатываемое событие по политике. Невалидный запрос не маскируется ACCEPTED. |
| Доставка | PENDING, IN_FLIGHT, RETRY_WAIT, DELIVERED, FAILED, CANCELLED | Успех терминален. FAILED может быть вручную возвращён в PENDING с аудитом. CANCELLED не исполняется. |

Основные переходы Delivery: `PENDING / RETRY_WAIT → IN_FLIGHT → DELIVERED / RETRY_WAIT / FAILED`. При отмене — CANCELLED; при истечении lease незавершённая попытка отмечается UNKNOWN, доставка получает повтор с прежним ID. UNKNOWN — исход попытки, не доказательство, что получатель ничего не сделал.

## Контракты и безопасность

### Входящие сообщения

Предлагаемый route: `POST /webhooks/v1/{connection_id}`. API Gateway проксирует raw body и разрешённые signing headers в Webhook HTTP Handler **без JSON parse/reserialize, grpc-gateway транскодирования и пользовательских cookie**. Это отдельное требование к маршруту gateway, не утверждение о его уже реализованной конфигурации. Лимиты и access logs для этого маршрута не раскрывают body/секреты.

Для новых совместимых подключений предлагается схема [Standard Webhooks](https://github.com/standard-webhooks/standard-webhooks/blob/main/spec/standard-webhooks.md): заголовки `webhook-id`, `webhook-timestamp`, `webhook-signature`; HMAC-SHA256 от `message_id.timestamp.raw_body`, подпись `v1,<base64>`. Сравнение — constant-time, timestamp проверяется на свежесть. ID сохраняется при повторе, timestamp попытки обновляется. Для provider с другим протоколом нужен явно выбранный адаптер, не эвристика по входному payload.

Предлагаемое окно clock skew — 5 минут в обе стороны. Проверка свежести защищает от replay записанного запроса; долговременная SQL-дедупликация защищает от повторной доставки с новой корректной подписью. Повтор с устаревшей подписью не становится допустимым только потому, что ID уже есть в БД. Timestamp попытки и occurredAt бизнес-события — разные данные.

Секреты хранятся зашифрованными с отдельным deployment-managed ключом шифрования вне БД; версия ключа сохраняется для ротации. Нужны безопасная передача signing secret, audit и отсутствие секретов в logs/config exports. Secret manager/KMS — выбор deployment, новый контейнер не добавлен без согласованной инфраструктуры. Утечка signing key требует экстренного revoke, не только обычного grace period.

Предлагаемые ответы HTTP приёмника:

| HTTP status | Значение |
|---|---|
| 202 | Receipt зафиксирован; response содержит receipt ID и ACCEPTED/IGNORED. Корректный повтор возвращает исходный receipt, не новое событие. |
| 400 / 415 / 422 | Невалидная форма запроса, неподдерживаемый content type/encoding либо недопустимые schema/type/tenant данные. |
| 401 / 404 | Недопустимая подпись/свежесть либо неизвестное/отозванное подключение; response не раскрывает secrets. |
| 409 | Уже принятый message ID повторно использован с другим payload. |
| 413 / 429 | Превышен размер тела либо ограничение частоты. |
| 503 | Подключение временно PAUSED или приём нельзя надёжно зафиксировать; разрешён повтор с тем же message ID. |

### Исходящий payload

Предлагаемый JSON envelope:

```json
{
  "id": "delivery_id",
  "type": "hiring.application.stage_changed",
  "schemaVersion": 1,
  "occurredAt": "2026-10-08T12:00:00Z",
  "sourceEventId": "domain_event_id",
  "data": {
    "applicationId": "application_id",
    "stage": "INTERVIEW"
  }
}
```

Это иллюстрация контракта, не существующая схема. Поля data задаются allowlist конкретного типа; внешний получатель не получает весь внутренний Avro event, скрытые контакты или файл резюме. Сервис не дополняет payload прямым SQL-запросом к чужой БД. Если событию не хватает необходимых данных, уточняется контракт владельца или отдельный разрешённый API — зависимость не выдумывается молча.

Webhook ID равен delivery ID, source event ID остаётся отдельным полем. Из одного внутреннего события может возникнуть несколько доставок разных подписок. На каждой попытке подписываются сохранённые bytes с новым attempt timestamp и действующим ключом; ID/occurredAt/data не меняются. Для проверки endpoint используется отдельный test delivery, не повтор реального бизнес-события.

### Защита исходящих адресов

Webhook destination — потенциальный SSRF-вход. Требуются HTTPS, разрешённые hosts/ports, запрет credentials в URL и redirects. Проверяются IPv4/IPv6 и итоговые адреса, включая loopback, private/link-local, metadata и сети Careeria. HTTP client соединяется только с проверенным адресом, сохраняя исходное имя для TLS/SNI; повторная непроверенная DNS resolution недопустима. Политика применяется к каждой попытке и fallback, дополняется egress-ограничениями. Основание: [OWASP SSRF Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html).

В первой версии предлагается регистрация структурированных host/path и фиксированного HTTPS/443 для разрешённых интеграций; произвольные схемы/порты и внутренние HR endpoints не поддерживаются. Если нужен private endpoint, требуется отдельный согласованный deployment path, а не выключение SSRF-проверок. Тестовое подтверждение адреса проходит тот же защищённый HTTP client. Внешний response не используется как новая destination или исполняемый mapping.

## Каталог сценариев

| № | Сценарий | Прикладной компонент | Результат |
|---|---|---|---|
| 1 | CreateIntegrationConnection | Management | Подключение компании и параметры входящего route. |
| 2 | UpdateIntegrationConnection | Management | Разрешённые настройки и mapping revision обновлены. |
| 3 | Pause / ResumeIntegrationConnection | Management | Новая обработка/доставка остановлена либо возобновлена. |
| 4 | RevokeIntegrationConnection | Management | Подключение и credentials отозваны. |
| 5 | RotateSigningKey | Management | Новая версия ключа с контролируемым переходом. |
| 6 | CreateWebhookSubscription | Management | Неподтверждённая исходящая подписка. |
| 7 | VerifyWebhookEndpoint | Management | Контролируемый test delivery и подтверждение получателя. |
| 8 | UpdateWebhookSubscription | Management | Фильтры/внешний контракт/endpoint revision изменены. |
| 9 | Pause / Resume / DeleteWebhookSubscription | Management | Управление исходящей подпиской. |
| 10 | ReceiveInboundWebhook | Inbound | Проверенный receipt и нормализованный outbox event. |
| 11 | HandleDuplicateInboundWebhook | Inbound | Исходный receipt без повторного бизнес-события. |
| 12 | IgnoreUnsupportedNotification | Inbound | Устойчиво зафиксированное игнорирование известного типа. |
| 13 | PlanOutboundDeliveries | Outbound | Все разрешённые доставки внутреннего события записаны. |
| 14 | ExecuteWebhookDelivery | Outbound / worker | Одна контролируемая HTTP-попытка и её результат. |
| 15 | RetryWebhookDelivery | Outbound / worker | Отложенный повтор с тем же delivery ID. |
| 16 | RecoverExpiredDeliveryLease | Outbound / worker | Продолжение после падения процесса без потери job. |
| 17 | ListConnections / Subscriptions | Management | Ограниченный список конфигурации своей компании. |
| 18 | Get / ListInboundReceipts | Management | Журнал приёма без ложного статуса «применено». |
| 19 | Get / ListDeliveries / Attempts | Management | История исходящих попыток и ошибок. |
| 20 | RedriveFailedDelivery | Management | Разрешённый ручной повтор FAILED-доставки. |
| 21 | CancelWebhookDelivery | Management | Новые попытки запрещены, результат текущей не скрывается. |

### 1. CreateIntegrationConnection

**Вход:** company ID, поддерживаемый provider type, external tenant reference, allowlist входящих событий и mapping version. Пользователь берётся из доверенного identity context. Management проверяет право управления интеграциями именно этой компании, продуктовые лимиты и допустимость provider/контракта.

Создаются connection ID, inbound route и отдельный криптографически случайный inbound signing key. Состояние ACTIVE означает, что настроенный источник может отправлять подписанные сообщения; исходящая подписка создаётся отдельно. Настройки, encrypted key, idempotency record и audit фиксируются атомарно. Signing secret выдаётся один раз по защищённому каналу; GET возвращает только masked metadata/key version.

Повтор исходной create-команды должен вернуть тот же connection ID. Если одноразовая выдача секрета была потеряна, не предполагается его повторное раскрытие из общего журнала: требуется безопасный протокол выдачи или RotateSigningKey. company ID нельзя менять обычным Update.

### 2. UpdateIntegrationConnection

**Вход:** connection ID, expected version и field mask. Проверяются права компании, не-REVOKED состояние и совместимость mapping/allowed event types. Маппинг — зарегистрированный версионированный код/конфигурация, не загруженный пользователем скрипт или произвольная URL для исполнения.

Транзакция сохраняет новую configuration version и audit. Уже принятый receipt содержит прежнюю mapping/schema version и не перенормализуется автоматически. Изменение external tenant binding — security-sensitive операция: предлагается запрет обычного patch, создание нового подключения вместо незаметного переноса источника.

### 3. Pause / ResumeIntegrationConnection

Владелец полномочий переводит ACTIVE ↔ PAUSED доменным методом с expected version. PAUSED блокирует новый webhook приём, новые исходящие deliveries и дальнейший запуск jobs; существующие RETRY_WAIT сохраняются до resume или общего delivery deadline. Входящему provider возвращается временная ошибка 503 с ограниченным Retry-After, чтобы он мог повторить сообщение.

Resume не оживляет REVOKED, не подтверждает PENDING_VERIFICATION подписки и не возвращает отменённые deliveries. Новая попытка дополнительно проверяет актуальный статус connection непосредственно перед сетью. Уже отправленный HTTP-запрос нельзя отозвать SQL-изменением; pause не обещает мгновенный rollback действия внешней системы.

### 4. RevokeIntegrationConnection

Команда требует расширенного management permission, expected version и аудита. Подключение становится REVOKED, signing keys отзываются, подписки деактивируются. Неисполненные deliveries этого подключения должны стать CANCELLED; для большого backlog допускается порционная обработка, но scheduler/worker сразу проверяет connection status и не ждёт массового update.

Старый route не принимает сообщения, resume запрещён. Receipt/attempt history сохраняется по retention; revoke не стирает применённые изменения в Hiring и не отзывает уже завершённый внешний callback. Повтор отзыва — идемпотентный результат.

### 5. RotateSigningKey

**Вход:** connection/subscription ID, direction, expected version и режим normal/emergency. Для входа и выхода ротация независима. Создаётся encrypted новая версия, задаётся срок перехода; предыдущая временно принимается/используется только по согласованной политике provider. Для совместимого протокола можно отправлять несколько подписей в одном header.

Новые attempts используют текущие разрешённые ключи, а журнал хранит только их version. По окончании окна retiring key перестаёт приниматься. Emergency rotation отзывает старый ключ без grace и требует координации с получателем/отправителем; нельзя продолжать подписывать скомпрометированным ключом ради успешного retry. Потеря deployment encryption key блокирует операции с секретами и требует отдельного восстановления, не fail-open.

### 6. CreateWebhookSubscription

**Вход:** connection ID, структурированный destination host/path, event allowlist и payload schema version. Management проверяет компанию, ACTIVE/PAUSED connection, разрешённый внешний контракт, destination policy и лимиты подписок. Пользователь не может подписаться на любые Kafka topics или указать чужой company ID.

Создаются subscription ID/revision, отдельный outbound key и состояние PENDING_VERIFICATION. Реальные бизнес-события ещё не отправляются. Уникальные ограничения и idempotency key предотвращают дубли после потерянного ответа. Конфигурация не содержит файлов резюме или секретов другого подключения.

### 7. VerifyWebhookEndpoint

В первой версии предлагается challenge handshake: Management генерирует одноразовый challenge с TTL и сохраняет **test delivery**; worker доставляет его с той же сетевой политикой и подписью. Получатель возвращает согласованный challenge response, а сервис сверяет его по ограниченному ответу. Один HTTP 200 от произвольного сайта не считается доказательством контроля endpoint.

Outbound Webhook Service обрабатывает результат test delivery и транзакционно активирует только ту subscription revision, для которой был создан challenge; Management инициирует проверку и показывает её результат. Просроченный/повторный или пришедший после изменения destination ответ не подтверждает новый адрес. Нужны точная JSON-схема handshake и provider-specific альтернативы; они ещё не согласованы. Отправка теста — предложенный пользовательский сценарий, не действие, которое выполняется при проектировании документации.

### 8. UpdateWebhookSubscription

**Вход:** subscription ID, expected version, разрешённые фильтры и schema version. Меняется revision, сохраняется audit. Изменение destination требует новой verification и блокирует дальнейшие отправки прежней revision; ещё не отправленные jobs старого адреса отменяются. Реальные payload snapshots не переписываются новым контрактом.

Зарегистрированная смена секретов выполняется RotateSigningKey, не случайным patch. Уже IN_FLIGHT delivery может закончиться на старом адресе; это отражается в attempt history. Перенос всего backlog на новый host не происходит автоматически — отдельная миграционная операция вне первой версии.

### 9. Pause / Resume / DeleteWebhookSubscription

После проверки прав PAUSED подписка перестаёт получать новые jobs и выполнять старые; pending jobs сохраняются в пределах общего deadline. После resume отправляются только разрешённые непросроченные jobs, без восстановления пропущенных событий периода pause. Для backfill нужен отдельный контракт с источником, не обещание перечитать произвольный Kafka history.

DELETED подписка терминальна, не планирует deliveries; дальнейшие jobs отменяются/блокируются. Журнал и source event IDs не удаляются немедленно. Resume не активирует неподтверждённый или удалённый endpoint.

### 10. ReceiveInboundWebhook

**Вход:** POST, connection ID из route, raw bytes, signing headers и request metadata. Сервис сначала применяет лимиты, находит допустимое подключение, проверяет подпись/clock skew, затем разбирает schema. company ID и допустимый external tenant определяются сохранённым connection binding; противоречащие ему поля payload отклоняются.

Inbound Webhook Adapter нормализует только разрешённый тип события и разрешённые поля. В одной транзакции создаются InboundReceipt с digest/сообщением и нормализованный outbox event. Provider message ID и local normalized event ID сохраняются неизменными. После commit возвращается 202 с receipt ID; publisher/consumer могут быть временно недоступны, принятый webhook не теряется.

Если SQL commit неизвестен из-за обрыва связи, ответ не утверждает успешный приём: отправитель повторяет тот же message ID, а уникальный индекс восстанавливает исходный receipt. Service не держит транзакцию до применения Hiring Service. Ограничение body — предлагаемо 1 MiB; compressed body в первой версии не принимается без отдельной byte/signature политики. Сырой payload, если нужен для расследования, хранится ограниченно и зашифрованно, не в access logs.

### 11. HandleDuplicateInboundWebhook

Подпись и freshness проверяются и для повторного запроса. При совпадении connection ID + provider message ID и digest возвращается тот же receipt без второй outbox-записи. Конкурентные POST защищает unique constraint; оба не могут одновременно создать независимые бизнес-события.

Тот же ID с другим body/digest — конфликт 409 и audit сигнал; он не заменяет исходное сообщение. Если provider не предоставляет стабильный идентификатор, требуется согласованный provider-specific ключ. Хеш payload нельзя молча использовать вместо event ID: два одинаковых по содержимому бизнес-события могут быть разными.

### 12. IgnoreUnsupportedNotification

Некоторые provider отправляют типы, явно помеченные политикой подключения «известный, игнорируемый». После проверки подписи/tenant создаётся IGNORED receipt с безопасной причиной, без доменного outbox, отправителю возвращается 202 с receipt и status=IGNORED. Повтор получает тот же результат.

Неизвестная schema, невалидные данные или запрещённый tenant не скрываются успешным ignore. Неподдерживаемый бизнес-тип возвращает 422, если не внесён в явную ignore policy. Неподлинные запросы не записываются как доверенные события; сохраняется только ограниченный security audit.

### 13. PlanOutboundDeliveries

Events Consumer декодирует внутреннее Avro-событие, проверяет тип/schema и передаёт source event ID, trusted company ID и payload Outbound Service. Источник — разрешённый внутренний topic/контракт, а не подписанный внешним клиентом запрос, автоматически признанный доменным событием.

Исходящий consumer подписан на бизнес-события сервисов-владельцев, а не на собственный `careeria.integrations.inbound.v1`. Входящая нормализованная команда не отправляется внешним подписчикам до её проверки и применения доменом.

В одной SQL-транзакции проверяется ProcessedEvent и создаются deliveries для подходящих ACTIVE connections/subscriptions данной компании. Для каждой фиксируются endpoint revision, согласованная внешняя schema и точные payload bytes. Затем записывается ProcessedEvent; только после commit consumer подтверждает offset. Падение до offset commit приводит к повторному чтению без второй группы jobs, благодаря processed event и unique subscription + source event.

Отсутствие подписчиков — устойчивый успешный результат, не бесконечный retry. Число подписок/размер fan-out ограничены; большой fan-out потребует отдельного устойчивого planning job вместо одной неограниченной транзакции. Невалидная/неизвестная schema сохраняется в карантин с причиной до подтверждения offset; техническая ошибка БД не подтверждает сообщение. Требуется определить retention и операторский redrive карантина, а не добавить Redis «для дедупликации» без контракта.

### 14. ExecuteWebhookDelivery

Worker вызывает прикладной сценарий захвата due jobs. Короткая транзакция с `FOR UPDATE SKIP LOCKED` переводит delivery в IN_FLIGHT, выдаёт уникальный lease token и создаёт STARTED attempt; commit завершается **до** сетевого вызова. Сервис проверяет актуальный connection/subscription, revision, deadline, разрешённый destination и ключи.

Outbound Webhook Adapter подписывает неизменяемые bytes с текущим attempt timestamp. HTTP client делает один HTTPS POST с ограничением времени и ответа; отключены redirects и неучтённые автоматические POST retries. Каждая реальная попытка должна соответствовать Attempt, а не скрытому циклу библиотеки.

После ответа отдельная транзакция с проверкой lease token сохраняет outcome и обновляет Delivery. 2xx означает DELIVERED по соглашению транспорта — не успешное выполнение всех бизнес-операций получателем. Timeout означает UNKNOWN результат попытки: получатель мог применить сообщение. Поэтому следующая попытка сохраняет webhook ID, и внешний получатель обязан дедуплицировать.

### 15. RetryWebhookDelivery

Предлагаемая политика: exponential backoff с jitter, bounded attempts и общий delivery deadline. 429 учитывает ограниченный Retry-After; 408, 5xx и сетевые сбои повторяются в пределах бюджета. Большинство остальных 4xx и 3xx — FAILED, без redirect/fail-open. 401/403 требуют проверки ключа/конфигурации оператором, не бессмысленного мгновенного бесконечного retry.

Retry-After не позволяет отложить job бессрочно за deadline. Следующая попытка имеет новый attempt ID/timestamp, но тот же delivery ID/source event/payload. В первой версии предлагается до 10 attempts и до 24 часов доставки; значения требуют продуктового согласования. Destination/permission/status проверяются повторно, даже если предыдущая попытка была разрешена.

### 16. RecoverExpiredDeliveryLease

Если worker погиб, другой процесс после lease expiry возвращает job в retry path и отмечает незавершённый attempt UNKNOWN. Lease длиннее максимального request timeout с запасом либо продлевается с контролируемым heartbeat. Старый worker не завершает job нового владельца: SQL update требует lease token.

Одновременная физическая отправка всё же возможна при сетевой неопределённости/паузе процесса; lease не превращает внешний HTTP в exactly-once. Внешняя дедупликация стабильного webhook ID — часть контракта. Просроченная доставка становится FAILED, а revoked/deleted ресурс — CANCELLED.

### 17–19. Просмотр конфигурации и журналов

Management проверяет company/resource permission на каждом Get/List. Cursor pagination имеет bounded page size, фильтры по connection/subscription, времени и статусу; параметры не позволяют читать чужие deliveries по произвольному ID. Secret, подписи и полные payload по умолчанию отсутствуют в ответе.

Inbound journal показывает receipt, provider message ID, receivedAt и ACCEPTED/IGNORED. ACCEPTED не переименовывается в APPLIED на основании Kafka publish. Для отображения результата домена нужен отдельный коррелированный outcome event от владельца и согласованный контракт.

Outbound journal показывает delivery ID, source event ID, snapshot revision, attempts, timestamps, HTTP status/error class и nextAttemptAt. UNKNOWN отличается от доказанного HTTP-отказа. Просмотр содержимого чувствительного payload требует отдельного permission/audit; body retention и дедупликационный retention не обязаны совпадать.

### 20. RedriveFailedDelivery

**Вход:** delivery ID, expected version и обязательная причина. Management проверяет расширенные полномочия, FAILED, допустимый неизменённый endpoint revision и ACTIVE подтверждённую subscription/connection. Затем атомарно переводит delivery в PENDING с отдельным redrive generation/budget и audit, сохраняя все прошлые attempts и тот же webhook ID/payload.

DELIVERED не отправляется повторно этим RPC; IN_FLIGHT не получает параллельный job. Новый endpoint не подменяет старый snapshot молча. Ручной retry не гарантирует повторное бизнес-действие: если получатель уже обработал ID при потерянном ответе, он вернёт свой идемпотентный результат. Повтор redrive-команды защищён ключом идемпотентности.

### 21. CancelWebhookDelivery

Management после авторизации помечает PENDING/RETRY_WAIT как CANCELLED. Для IN_FLIGHT фиксируется cancelRequested: новые попытки запрещены, но текущий внешний запрос может завершиться. Ответ отражает неопределённость, а не обещает, что сообщение не достигло адресата.

Завершившийся 2xx сохраняет DELIVERED с информацией о запросе отмены; ошибка/UNKNOWN после отмены не запускает новый retry. Уже DELIVERED невозможно «отменить» удалением строки. Отмена доставки не откатывает состояние Careeria или внешней HR-системы.

## Дополнительные правила надёжности

### Раздельные гарантии

Приём обеспечивает один устойчивый нормализованный event на connection/message ID. CDC и Kafka могут доставлять повторно; доменный consumer дедуплицирует normalized event ID в своей транзакции. Outbound planner обеспечивает одну Delivery на subscription/source event, но её HTTP-исполнение — at-least-once в пределах retry budget, а не гарантированное вечное ожидание до успеха.

В первой версии глобальный порядок callbacks не обещается. Payload должен содержать aggregate ID/version, если тип события требует защиты от устаревших изменений. Строгая последовательность по ресурсу потребует отдельного dispatch policy и блокирует следующие сообщения при отказе предыдущего; её нельзя считать обеспеченной одними Kafka partition offsets, когда jobs выполняются параллельно.

### Циклы внешней синхронизации

Входящий webhook сохраняет source connection и correlation/causation ID. При доменном применении эти данные должны попасть в итоговое событие владельца. Outbound Service по умолчанию не отправляет отражённое изменение обратно в ту же source connection; другие разрешённые получатели могут его получить. Ограничение выполняется по доверенной provenance, не заголовку, заявленному чужим клиентом.

Если доменный сервис не переносит происхождение изменения, loop suppression невозможна надёжно — нужно уточнить контракт события. Сервис интеграций не должен угадывать происхождение по совпадающему JSON или времени. External entity mappings также не обходят company ownership и FSM Hiring Service.

### События и retention

Нормализованные входящие сообщения публикуются в предлагаемом topic `careeria.integrations.inbound.v1` с envelope: normalized event ID, connection/company ID, source/provider message ID, schema/mapping version, occurredAt/receivedAt и разрешённый payload. Это доверенная интеграционная команда/наблюдение, не событие «отклик уже изменён». Schema регистрируется deployment/CI-процессом; consumer только читает её из Registry.

Технические сообщения о FAILED доставке могут стать отдельными outbox events для мониторинга/уведомлений, но endpoint subscriptions не должны автоматически слушать собственные delivery-status events и создавать бесконечный webhook loop. На текущей L3 outbox показан для входящего нормализованного пути; расширение технических событий требует отдельного allowlist.

Сроки хранения inbox IDs, ProcessedEvent и уникальных delivery keys должны покрывать provider retries, Kafka replay и административные повторы. Очистка payload раньше IDs допустима; удаление дедупликационных ключей вместе с коротким body retention может повторно выполнить старое сообщение. Предлагаемые сроки ещё не утверждены. Бесконечное хранение raw HR payload не является способом обеспечения идемпотентности.

### Наблюдаемость и проверки реализации

Нужны метрики signature failures, duplicate/conflicting incoming IDs, intake latency, oldest pending delivery, retry/FAILED counts, consumer lag, lease recoveries и per-endpoint success rate. Logs/audit содержат correlation/delivery/receipt ID, actor, company и безопасный error class, не secrets и тела HR-сообщений. Rate limits задаются на connection, endpoint и worker concurrency.

При реализации обязательны тесты: изменение одного байта подписи; timestamp прошлого/будущего; две конкурентные одинаковые входящие команды; тот же ID с другим body; crash после SQL commit до ответа/offset; timeout после применения получателем; crash worker и устаревший lease; key rotation/revoke; pause/delete во время отправки; DNS rebinding/IPv6/private destination/redirect; replay внутренних событий; tenant isolation; запрет отражения сообщения обратно источнику. Документирование этих тестов не означает, что backend уже реализован и прошёл их.

## Требует согласования перед реализацией

| Вопрос | Предлагаемая граница |
|---|---|
| Провайдеры и mappings | Начать с generic совместимого webhook и одного HR provider; не универсальный исполняемый ETL. |
| Внешние event schemas | Версионированные JSON/OpenAPI-контракты и allowlist полей по каждому бизнес-типу. |
| Endpoint verification | Challenge response, TTL, provider exceptions и разрешённые destinations. |
| Raw HTTP gateway route | Сохранение подписанных bytes/headers без пользовательского auth/transcoding; лимиты и защищённая передача в сервис. |
| Управление секретами | Deployment encryption key/KMS, одноразовая выдача, normal/emergency rotation и audit. |
| Retry, retention и порядок | Лимиты/сроки, redrive budget, dedup horizon, последовательность по aggregate при необходимости. |
| Исходные и итоговые доменные события | Company ID, aggregate version, provenance; owner проверяет ресурс и FSM, не доверяет webhook автоматически. |
| Recovery/quarantine | Операторский redrive невалидных внутренних сообщений, наблюдаемость и сохранение причины до offset commit. |
| Не-webhook интеграции | Polling стороннего REST API, импорт файлов, SFTP и сторонние SDK не включены в первую версию. |

[Все L3](../README.md) · [Платформенные контейнеры · L2](../../projections/members/member4/member4-l2-containers.md) · [Приём HR-webhook](../../dynamic/hr-integration.md) · [Hiring Service](hiring-service.md)
