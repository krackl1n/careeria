# 3.3. Файлы, уведомления и внешние интеграции

## File Service

Централизованный сервис для безопасной работы с объектным хранилищем **S3 / MinIO**.

**Функции:**

- выдача краткоживущих presigned URL и поддержка multipart upload;
- проверка размера, MIME-типа, антивирусное сканирование и управление жизненным циклом объектов;
- хранение резюме, аватаров, логотипов компаний, вложений чатов, файлов заданий, портфолио и записей интервью.

Доменные сервисы хранят в своих БД только идентификатор `file_id` (например, `Resume { id, candidate_id, file_id }`), а бинарные данные размещаются в MinIO.

---

## Notification Service

Асинхронно потребляет доменные события из Kafka (например, `InterviewScheduled`, `ApplicationStageChanged`) и доставляет уведомления по каналам:

- **In-app** (внутри интерфейса платформы);
- **Email** (через SendGrid);
- **Push** (через Firebase Cloud Messaging);
- **SMS** (через Twilio).

Компонентная схема сервиса: [Notification Service — компоненты L3](../../architecture/c4-plant-uml/plantuml/component/backend/notification-service.md).

---

## Integration Service

Обеспечивает двустороннюю интеграцию Careeria с внешними корпоративными HR-системами работодателей:

- принимает входящие подписанные HTTPS webhooks с проверкой идемпотентности и сохраняет их в транзакционный inbox/outbox;
- доставляет исходящие webhook-события внешним системам с lease-блокировками и политикой повторных попыток (retries).

См. также:

- [Integration Service — компоненты L3](../../architecture/c4-plant-uml/plantuml/component/backend/integration-service.md)
- [Сценарий обработки HR-webhook](../../architecture/c4-plant-uml/plantuml/dynamic/hr-integration.md)

