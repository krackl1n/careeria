# Обработка HR-webhook

Внешняя HR-система передаёт изменение в Careeria; изменение состояния найма выполняет Hiring Service.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/dynamic/hr-integration.puml
@enduml
```

## Порядок действий

1. Employer HR System отправляет webhook через HTTPS в API Gateway.
2. Gateway передаёт исходное тело и signing headers по отдельному HTTP route без JSON-пересериализации и пользовательской сессии.
3. Integration Service проверяет подключение, подпись, свежесть, tenant и payload. Нормализует допустимое сообщение и атомарно сохраняет inbox, ключ дедупликации и outbox в Integration DB. После commit возвращает 202 с receipt ID; повтор исходного message ID не создаёт второе событие.
4. Debezium читает Integration outbox через PostgreSQL WAL.
5. Debezium публикует нормализованное событие в Kafka по согласованной Avro schema.
6. Hiring Service получает событие. Оно ещё не означает, что бизнес-изменение допустимо.
7. Hiring Service проверяет источник/компанию, mapping ресурса, версию и локальную FSM; сохраняет изменение, историю, дедупликацию и свой outbox в одной транзакции. Offset подтверждает после фиксации.

202 означает надёжный приём, а не успешное изменение отклика. Техническая повторная доставка сохраняет нормализованный event ID; доменный consumer обрабатывает её идемпотентно. Запрещённый переход не записывается напрямую в Hiring DB. Для обратной исходящей доставки нужен отдельный доменный event после применения; его происхождение позволяет избежать отражения изменения обратно тому же источнику.

[Integration Service · L3](../component/backend/integration-service.md) · [Hiring Service · L3](../component/backend/hiring-service.md)
