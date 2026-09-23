candidate -> platform "Ищет вакансии и откликается" "HTTPS"
recruiter -> platform "Управляет вакансиями и кандидатами" "HTTPS"
hiringManager -> platform "Оценивает кандидатов" "HTTPS"
companyAdmin -> platform "Управляет компанией и ролями" "HTTPS"

platform -> identityProvider "Аутентифицирует пользователей" "OAuth 2.0 / OIDC"
platform -> notificationProviders "Доставляет уведомления" "Email / Push / SMS"
platform -> employerHrSystem "Передаёт данные найма" "HTTPS / REST / Webhooks"
employerHrSystem -> platform "Передаёт изменения" "HTTPS / Webhooks"

candidate -> platform.web "Использует" "HTTPS"
recruiter -> platform.web "Использует" "HTTPS"
hiringManager -> platform.web "Использует" "HTTPS"
companyAdmin -> platform.web "Использует" "HTTPS"

platform.web -> platform.bff "Вызывает API" "HTTPS / JSON"
platform.bff -> platform.apiGateway "Вызывает API" "HTTPS / gRPC"

platform.apiGateway -> platform.identityService "Проверяет пользователя" "gRPC"
platform.identityService -> identityProvider "Аутентифицирует пользователя" "OAuth 2.0 / OIDC"

platform.apiGateway -> platform.candidateService "Получает профили кандидатов" "gRPC"
platform.apiGateway -> platform.companyService "Управляет компаниями" "gRPC"
platform.apiGateway -> platform.vacancyService "Управляет вакансиями" "gRPC"
platform.apiGateway -> platform.hiringService "Ведёт процесс найма" "gRPC"
platform.apiGateway -> platform.assessmentService "Управляет оцениванием" "gRPC"
platform.apiGateway -> platform.communicationService "Обменивается сообщениями" "gRPC"
platform.apiGateway -> platform.notificationService "Управляет уведомлениями" "gRPC"
platform.apiGateway -> platform.fileService "Работает с файлами" "gRPC"
platform.apiGateway -> platform.integrationService "Передаёт webhook" "HTTP / gRPC"

employerHrSystem -> platform.apiGateway "Отправляет webhook" "HTTPS / JSON"
platform.integrationService -> employerHrSystem "Передаёт события найма" "HTTPS / REST / Webhooks"

platform.candidateService -> platform.openFga "Проверяет доступ" "OpenFGA API" {
    tags "InfrastructureConnection"
}
platform.companyService -> platform.openFga "Проверяет доступ" "OpenFGA API" {
    tags "InfrastructureConnection"
}
platform.vacancyService -> platform.openFga "Проверяет доступ" "OpenFGA API" {
    tags "InfrastructureConnection"
}
platform.hiringService -> platform.openFga "Проверяет доступ" "OpenFGA API" {
    tags "InfrastructureConnection"
}
platform.assessmentService -> platform.openFga "Проверяет доступ" "OpenFGA API" {
    tags "InfrastructureConnection"
}
platform.communicationService -> platform.openFga "Проверяет доступ" "OpenFGA API" {
    tags "InfrastructureConnection"
}
platform.fileService -> platform.openFga "Проверяет доступ" "OpenFGA API" {
    tags "InfrastructureConnection"
}

platform.candidateService -> platform.candidateDb "Хранит профили и резюме" "SQL"
platform.companyService -> platform.companyDb "Хранит компании и роли" "SQL"
platform.vacancyService -> platform.vacancyDb "Хранит вакансии" "SQL"
platform.hiringService -> platform.hiringDb "Хранит отклики и этапы" "SQL"
platform.assessmentService -> platform.assessmentDb "Хранит задания и оценки" "SQL"
platform.communicationService -> platform.communicationDb "Хранит сообщения" "SQL"
platform.identityService -> platform.identityDb "Хранит учётные данные" "SQL"

platform.fileService -> platform.objectStorage "Читает и записывает файлы" "S3 API"

platform.candidateDb -> platform.debezium "Передаёт outbox" "PostgreSQL WAL" {
    tags "CDC,InfrastructureConnection"
}

platform.companyDb -> platform.debezium "Передаёт outbox" "PostgreSQL WAL" {
    tags "CDC,InfrastructureConnection"
}

platform.vacancyDb -> platform.debezium "Передаёт outbox" "PostgreSQL WAL" {
    tags "CDC,InfrastructureConnection"
}

platform.hiringDb -> platform.debezium "Передаёт outbox" "PostgreSQL WAL" {
    tags "CDC,InfrastructureConnection"
}

platform.assessmentDb -> platform.debezium "Передаёт outbox" "PostgreSQL WAL" {
    tags "CDC,InfrastructureConnection"
}

platform.communicationDb -> platform.debezium "Передаёт outbox" "PostgreSQL WAL" {
    tags "CDC,InfrastructureConnection"
}

platform.identityDb -> platform.debezium "Передаёт outbox" "PostgreSQL WAL" {
    tags "CDC,InfrastructureConnection"
}

platform.debezium -> platform.kafka "Публикует события" "Kafka / Avro" {
    tags "Async"
}

platform.debezium -> platform.schemaRegistry "Получает схемы" "Schema Registry API / Avro" {
    tags "InfrastructureConnection"
}

platform.kafka -> platform.hiringService "Доставляет события найма" "Kafka / Avro" {
    tags "Async"
}

platform.kafka -> platform.notificationService "Доставляет события уведомлений" "Kafka / Avro" {
    tags "Async"
}

platform.kafka -> platform.realtimePublisher "Доставляет realtime-события" "Kafka / Avro" {
    tags "RealtimeFlow"
}

platform.kafka -> platform.authzProjection "Доставляет изменения ролей" "Kafka / Avro" {
    tags "Async"
}

platform.kafka -> platform.integrationService "Доставляет события интеграций" "Kafka / Avro" {
    tags "Async"
}

platform.integrationService -> platform.kafka "Публикует события" "Kafka / Avro" {
    tags "Async"
}

platform.hiringService -> platform.schemaRegistry "Получает схемы" "Schema Registry API / Avro" {
    tags "InfrastructureConnection"
}
platform.notificationService -> platform.schemaRegistry "Получает схемы" "Schema Registry API / Avro" {
    tags "InfrastructureConnection"
}
platform.realtimePublisher -> platform.schemaRegistry "Получает схемы" "Schema Registry API / Avro" {
    tags "InfrastructureConnection"
}
platform.authzProjection -> platform.schemaRegistry "Получает схемы" "Schema Registry API / Avro" {
    tags "InfrastructureConnection"
}
platform.integrationService -> platform.schemaRegistry "Управляет схемами" "Schema Registry API / Avro" {
    tags "InfrastructureConnection"
}

platform.authzProjection -> platform.openFga "Обновляет роли и связи" "OpenFGA API"

platform.realtimePublisher -> platform.centrifugo "Публикует сообщения" "Centrifugo API" {
    tags "RealtimeFlow"
}

platform.web -> platform.centrifugo "Подключается" "WebSocket" {
    tags "RealtimeFlow"
}

platform.centrifugo -> platform.web "Доставляет сообщения" "WebSocket" {
    tags "RealtimeFlow"
}

platform.hiringService -> platform.livekit "Создаёт комнаты" "LiveKit API" {
    tags "MediaFlow"
}

platform.web -> platform.livekit "Передаёт медиапоток" "WebRTC" {
    tags "MediaFlow"
}

platform.notificationService -> notificationProviders "Отправляет уведомления" "Email / Push / SMS"
