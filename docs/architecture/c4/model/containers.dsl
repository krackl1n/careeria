group "Точки входа" {
    web = container "Web Application" {
        description "App Router, SSR/CSR, TanStack Query, React Hook Form и Zod; интерфейс кандидатов и сотрудников компаний."
        technology "Next.js / React / TypeScript / Tailwind CSS"
        tags "Frontend"
    }

    bff = container "BFF" {
        description "HttpOnly-сессии, CSRF-защита, обновление токенов и агрегация ответов для Next.js."
        technology "Go / net/http / grpc-go / OIDC"
        tags "Edge"
    }

    apiGateway = container "API Gateway" {
        description "REST-to-gRPC маршрутизация, rate limiting, request ID, OpenTelemetry-трассировка и передача доверенного identity context."
        technology "Go / grpc-gateway / grpc-go / OpenTelemetry"
        tags "Gateway"
    }
}

group "Домен кандидатов" {
    candidateService = container "Candidate Service" {
        description "Профили кандидатов, резюме, опыт работы, образование, навыки и карьерные предпочтения."
        technology "Go / gRPC / Protobuf / pgx / sqlc"
        tags "BusinessService"
    }

    candidateDb = container "Candidate DB" {
        description "Данные Candidate Service и его транзакционный outbox."
        technology "PostgreSQL"
        tags "Database"
    }
}

group "Домен компаний" {
    companyService = container "Company Service" {
        description "Работодатели, участники компаний, рекрутеры и корпоративные роли."
        technology "Go / gRPC / Protobuf / pgx / sqlc"
        tags "BusinessService"
    }

    companyDb = container "Company DB" {
        description "Данные Company Service и его транзакционный outbox."
        technology "PostgreSQL"
        tags "Database"
    }
}

group "Домен вакансий" {
    vacancyService = container "Vacancy Service" {
        description "Вакансии, требования, условия работы, вознаграждение и жизненный цикл публикации."
        technology "Go / gRPC / Protobuf / pgx / sqlc"
        tags "BusinessService"
    }

    vacancyDb = container "Vacancy DB" {
        description "Данные Vacancy Service и его транзакционный outbox."
        technology "PostgreSQL"
        tags "Database"
    }
}

group "Домен найма" {
    hiringService = container "Hiring Service" {
        description "Отклики, воронки и этапы найма, собеседования, предложения о работе и состояния процесса найма."
        technology "Go / gRPC / Protobuf / pgx / sqlc"
        tags "BusinessService"
    }

    hiringDb = container "Hiring DB" {
        description "Данные Hiring Service и его транзакционный outbox."
        technology "PostgreSQL"
        tags "Database"
    }
}

group "Домен оценки" {
    assessmentService = container "Assessment Service" {
        description "Оценочные мероприятия, тесты, практические задания, Live Coding, решения и результаты проверки."
        technology "Go / gRPC / Protobuf / pgx / sqlc"
        tags "BusinessService"
    }

    assessmentDb = container "Assessment DB" {
        description "Данные Assessment Service и его транзакционный outbox."
        technology "PostgreSQL"
        tags "Database"
    }
}

group "Коммуникации" {
    communicationService = container "Communication Service" {
        description "Диалоги, участники, сообщения, вложения, реакции и отметки о прочтении."
        technology "Go / gRPC / Protobuf / pgx / sqlc"
        tags "BusinessService"
    }

    communicationDb = container "Communication DB" {
        description "Данные Communication Service и его транзакционный outbox."
        technology "PostgreSQL"
        tags "Database"
    }

    notificationService = container "Notification Service" {
        description "Формирует и доставляет уведомления в приложении, по электронной почте, через push и SMS."
        technology "Go / Kafka consumer / REST clients"
        tags "PlatformService"
    }
}

group "Идентификация и доступ" {
    identityService = container "Identity Service" {
        description "Регистрация, credentials, access/refresh tokens, ротация сессий и интеграция с Keycloak."
        technology "Go / gRPC / OAuth 2.0 / OIDC / JWT / Argon2id"
        tags "PlatformService"
    }

    identityDb = container "Identity DB" {
        description "Данные Identity Service и его транзакционный outbox."
        technology "PostgreSQL"
        tags "Database"
    }

    authzProjection = container "AuthZ Synchronization Service" {
        description "Консьюмер Kafka, который по доменным событиям изменяет роли и отношения доступа в OpenFGA."
        technology "Go / Kafka consumer / OpenFGA SDK"
        tags "PlatformService"
    }

    openFga = container "OpenFGA" {
        description "ReBAC-модель и tuples для проверок доступа к компаниям, вакансиям, откликам и интервью."
        technology "OpenFGA / PostgreSQL"
        tags "Authorization"
    }
}

group "Файлы" {
    fileService = container "File Service" {
        description "Multipart upload, presigned URL, MIME/size validation, antivirus hook и управление жизненным циклом."
        technology "Go / gRPC / MinIO Go SDK"
        tags "PlatformService"
    }

    objectStorage = container "Object Storage" {
        description "Резюме, аватары, вложения, файлы оценочных заданий и записи собеседований."
        technology "MinIO / S3 API"
        tags "ObjectStorage"
    }
}

group "Интеграции" {
    integrationService = container "Integration Service" {
        description "Проверяет подписи webhook, нормализует payload, публикует события и выполняет retry доставки."
        technology "Go / REST / Webhooks / Kafka"
        tags "PlatformService"
    }
}

group "Realtime и медиа" {
    realtimePublisher = container "Realtime Publisher" {
        description "Фильтрует события, формирует пользовательские каналы и публикует realtime-сообщения."
        technology "Go / Kafka consumer / Centrifugo HTTP API"
        tags "PlatformService,Infrastructure"
    }

    centrifugo = container "Centrifugo" {
        description "Управляет WebSocket-подключениями, каналами, presence и доставкой событий пользователям."
        technology "Centrifugo / WebSocket / JWT"
        tags "Realtime,Infrastructure"
    }

    livekit = container "LiveKit" {
        description "Комнаты интервью, аудио, видео, screen sharing и опциональная запись."
        technology "LiveKit / WebRTC / LiveKit React SDK"
        tags "Media,Infrastructure"
    }
}

group "Событийная платформа" {
    kafka = container "Kafka" {
        description "Топики доменных событий, consumer groups, retention и асинхронное взаимодействие предметных областей."
        technology "Apache Kafka / KRaft"
        tags "EventBus,Infrastructure"
    }

    schemaRegistry = container "Schema Registry" {
        description "Версионирование Avro-схем и backward compatibility для доменных событий."
        technology "Apicurio Registry / Avro"
        tags "EventInfrastructure,Infrastructure"
    }

    debezium = container "Debezium" {
        description "Читает transactional outbox через PostgreSQL logical replication и публикует события в Kafka."
        technology "Debezium PostgreSQL Connector"
        tags "EventInfrastructure,Infrastructure"
    }
}
