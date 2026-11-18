# Обработка HR-webhook

<!-- excerpt -->

Integration Service проверяет входящий вебхук работодателя и передаёт событие в Kafka. Hiring Service решает, допустимо ли запрошенное изменение этапа отклика.

Шлюз передаёт исходное тело запроса и заголовки подписи в Integration Service. После проверки сообщения сервис сохраняет запись приёма и нормализованное событие, затем подтверждает отправителю приём. Через Debezium и Kafka событие достигает Hiring Service. Изменение отклика выполняется только после проверки компании, ресурса и допустимого перехода его состояния.

```plantuml
@startuml
!include architecture/c4/plantuml/dynamic/hr-integration.puml
@enduml
```

<p align="center">Рисунок - Обработка HR-webhook</p>

<p align="right">Таблица - Элементы. Обработка HR-webhook</p>

| Элемент | Тип или технология | Назначение и ответственность |
|---|---|---|
| Employer HR System | Система | Кадровая система работодателя. Принимает результаты найма из Careeria и передаёт изменения кадровых данных через согласованные вебхуки. |
| API Gateway | Go, grpc-gateway, grpc-go | API Gateway принимает HTTP-запрос от BFF, проверяет предъявленный токен по открытым ключам Identity и применяет политику доступа к методу. Доменный сервис получает вызов по gRPC. |
| Integration Service | Go, HTTP, gRPC, Kafka, pgx, sqlc | Integration Service проверяет подпись входящего вебхука и передаёт принятое событие в Kafka. Для исходящих вебхуков он ведёт подписки работодателей и повторяет доставку при временном сбое. |
| Integration DB | PostgreSQL | Подключения, подписки, inbox, delivery jobs, попытки и transactional outbox. |
| Debezium | Debezium PostgreSQL Connector | Компонент захвата зафиксированных изменений PostgreSQL. Читает transactional outbox через журнал WAL и публикует доменные события в Kafka. |
| Kafka | Apache Kafka, KRaft | Событийная шина платформы. Хранит доменные сообщения в топиках и передаёт их группам потребителей для независимой асинхронной обработки. |
| Hiring Service | Go, gRPC, Protobuf, pgx, sqlc | Hiring Service ведёт отклик от первого этапа отбора до предложения о работе. Компоненты откликов, интервью и предложений согласуют переходы между этапами и сохраняют историю в Hiring DB. |
| Hiring DB | PostgreSQL | Applications, pipeline transitions, interviews, offers и transactional outbox. |

<p align="right">Таблица - Связи. Обработка HR-webhook</p>

| Источник | Получатель | Содержание взаимодействия | Протокол или способ вызова |
|---|---|---|---|
| Employer HR System | API Gateway | 1. Отправляет webhook. | HTTPS / JSON |
| API Gateway | Integration Service | 2. Передаёт сырое тело и подпись. | HTTP |
| Integration Service | Integration DB | 3. Проверяет и фиксирует inbox + outbox. Получатель выполняет чтение или запись в базе данных сервиса. | SQL transaction |
| Integration DB | Debezium | 4. Передаёт outbox. Коннектор читает зафиксированные изменения из журнала PostgreSQL. | PostgreSQL WAL |
| Debezium | Kafka | 5. Публикует нормализованное событие. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro |
| Kafka | Hiring Service | 6. Доставляет событие. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro |
| Hiring Service | Hiring DB | 7. Проверяет FSM и сохраняет изменение. Получатель выполняет чтение или запись в базе данных сервиса. | SQL transaction |

<!-- /excerpt -->

## Порядок действий

1. Employer HR System отправляет webhook через HTTPS в API Gateway.
2. Gateway передаёт исходное тело и signing headers по отдельному HTTP route без JSON-пересериализации и пользовательской сессии.
3. Integration Service проверяет подключение, подпись, свежесть, tenant и payload. Нормализует допустимое сообщение и атомарно сохраняет inbox, ключ дедупликации и outbox в Integration DB. После commit возвращает 202 с receipt ID; повтор исходного message ID не создаёт второе событие.
4. Debezium читает Integration outbox через PostgreSQL WAL.
5. Debezium публикует нормализованное событие в Kafka по согласованной Avro schema.
6. Hiring Service получает событие. Оно ещё не означает, что бизнес-изменение допустимо.
7. Hiring Service проверяет источник/компанию, mapping ресурса, версию и локальную FSM; сохраняет изменение, историю, дедупликацию и свой outbox в одной транзакции. Offset подтверждает после фиксации.

202 означает надёжный приём, а не успешное изменение отклика. Техническая повторная доставка сохраняет нормализованный event ID; доменный consumer обрабатывает её идемпотентно. Запрещённый переход не записывается напрямую в Hiring DB. Для обратной исходящей доставки нужен отдельный доменный event после применения; его происхождение позволяет избежать отражения изменения обратно тому же источнику.

[Integration Service · L3](../system-context%28L1%29/full-system%28L2%29/integration-service%28L3%29/index.md) · [Hiring Service · L3](../system-context%28L1%29/full-system%28L2%29/hiring-service%28L3%29/index.md)
