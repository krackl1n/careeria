# Обработка HR-webhook

Внешняя HR-система передаёт изменение в Careeria; изменение состояния найма выполняет Hiring Service.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/dynamic/hr-integration.puml
@enduml
```

## Порядок действий

1. Employer HR System отправляет webhook через HTTPS в API Gateway.
2. Gateway передаёт запрос Integration Service.
3. Integration Service проверяет подпись, нормализует данные и публикует событие Kafka.
4. Hiring Service получает событие.
5. Hiring Service сохраняет изменение в Hiring DB.

Обработчик webhook должен учитывать повторную доставку. Общие требования безопасности описаны в корневом README архитектуры.
