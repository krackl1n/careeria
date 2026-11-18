# Доставка доменного события

<!-- excerpt -->

Hiring Service сохраняет изменение и запись о событии в одной транзакции. Debezium публикует событие в Kafka, откуда его получает Notification Service для отправки уведомления.

Сценарий начинается после изменения процесса найма. Hiring Service фиксирует бизнес-данные и outbox, затем Debezium читает запись через журнал WAL. Kafka доставляет сообщение Notification Service, который выбирает способ уведомления и вызывает провайдера. Доставка сообщения пользователю выполняется через соответствующий канал email, push или SMS.

```plantuml
@startuml
!include architecture/c4/plantuml/dynamic/domain-event.puml
@enduml
```

<p align="center">Рисунок - Доставка доменного события</p>

<p align="right">Таблица - Элементы. Доставка доменного события</p>

| Элемент | Тип или технология | Назначение и ответственность |
|---|---|---|
| Notification Providers | Система | Внешние поставщики доставки сообщений. SendGrid обрабатывает email, Firebase Cloud Messaging доставляет push, Twilio отправляет SMS по запросам сервиса уведомлений. |
| Hiring Service | Go, gRPC, Protobuf, pgx, sqlc | Hiring Service ведёт отклик от первого этапа отбора до предложения о работе. Компоненты откликов, интервью и предложений согласуют переходы между этапами и сохраняют историю в Hiring DB. |
| Hiring DB | PostgreSQL | Данные и transactional outbox. |
| Debezium | Debezium PostgreSQL Connector | Компонент захвата зафиксированных изменений PostgreSQL. Читает transactional outbox через журнал WAL и публикует доменные события в Kafka. |
| Kafka | Apache Kafka, KRaft | Событийная шина платформы. Хранит доменные сообщения в топиках и передаёт их группам потребителей для независимой асинхронной обработки. |
| Notification Service | Go, Kafka consumer, REST clients | Notification Service принимает команды и события из Kafka, выбирает шаблон и создаёт задачу доставки. Обработчики сохраняют уведомления в личном кабинете или передают их провайдерам email, push и SMS. |

<p align="right">Таблица - Связи. Доставка доменного события</p>

| Источник | Получатель | Содержание взаимодействия | Протокол или способ вызова |
|---|---|---|---|
| Hiring Service | Hiring DB | 1. Записывает данные и outbox. Получатель выполняет чтение или запись в базе данных сервиса. | SQL |
| Hiring DB | Debezium | 2. Передаёт outbox. Коннектор читает зафиксированные изменения из журнала PostgreSQL. | PostgreSQL WAL |
| Debezium | Kafka | 3. Публикует событие. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro |
| Kafka | Notification Service | 4. Доставляет событие. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro |
| Notification Service | Notification Providers | 5. Отправляет уведомление. | Email / Push / SMS |

<!-- /excerpt -->

## Порядок действий

1. Hiring Service сохраняет данные и outbox в Hiring DB.
2. Debezium читает запись через PostgreSQL WAL.
3. Событие публикуется в Kafka в формате Avro.
4. Notification Service получает событие.
5. Внешний провайдер доставляет email, push или SMS.

Общий набор подписчиков показан в [событийной архитектуре](../event-driven%28L2%29/index.md).
