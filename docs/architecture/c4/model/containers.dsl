group "Точки входа" {
    web = container "Web Application" {
        description "Клиентское приложение для кандидатов и сотрудников компаний."
        technology "SPA"
        tags "Frontend"
    }

    bff = container "BFF" {
        description "Управляет пользовательской сессией, агрегирует данные сервисов и адаптирует API под веб-интерфейс."
        technology "HTTP API"
        tags "Edge"
    }

    apiGateway = container "API Gateway" {
        description "Единая точка входа во внутренние сервисы: маршрутизация, ограничение запросов, трассировка и передача доверенного контекста пользователя."
        technology "HTTP / gRPC"
        tags "Gateway"
    }
}

group "Домен кандидатов" {
    candidateService = container "Candidate Service" {
        description "Профили кандидатов, резюме, опыт работы, образование, навыки и карьерные предпочтения."
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
        tags "BusinessService"
    }

    communicationDb = container "Communication DB" {
        description "Данные Communication Service и его транзакционный outbox."
        technology "PostgreSQL"
        tags "Database"
    }

    notificationService = container "Notification Service" {
        description "Формирует и доставляет уведомления в приложении, по электронной почте, через push и SMS."
        tags "PlatformService"
    }
}

group "Идентификация и доступ" {
    identityService = container "Identity Service" {
        description "Регистрация, учётные данные, OAuth 2.0 и OpenID Connect, токены доступа и пользовательские сессии."
        tags "PlatformService"
    }

    identityDb = container "Identity DB" {
        description "Данные Identity Service и его транзакционный outbox."
        technology "PostgreSQL"
        tags "Database"
    }

    authzProjection = container "AuthZ Synchronization Service" {
        description "Консьюмер Kafka, который по доменным событиям изменяет роли и отношения доступа в OpenFGA."
        tags "PlatformService"
    }

    openFga = container "OpenFGA" {
        description "Источник истины для ролевой и основанной на отношениях авторизации."
        technology "OpenFGA"
        tags "Authorization"
    }
}

group "Файлы" {
    fileService = container "File Service" {
        description "Загрузка и скачивание файлов, метаданные, временные ссылки, проверка и управление жизненным циклом."
        tags "PlatformService"
    }

    objectStorage = container "Object Storage" {
        description "Резюме, аватары, вложения, файлы оценочных заданий и записи собеседований."
        technology "S3 / MinIO"
        tags "ObjectStorage"
    }
}

group "Интеграции" {
    integrationService = container "Integration Service" {
        description "Принимает и отправляет webhooks, преобразует внешние сообщения в события Careeria и управляет надёжной доставкой."
        tags "PlatformService"
    }
}

group "Realtime и медиа" {
    realtimePublisher = container "Realtime Publisher" {
        description "Получает события из Kafka, преобразует их для доставки в реальном времени и публикует в Centrifugo."
        tags "PlatformService,Infrastructure"
    }

    centrifugo = container "Centrifugo" {
        description "Публичная точка входа для WebSocket-соединений и доставки событий пользователям в реальном времени."
        technology "Centrifugo / WebSocket"
        tags "Realtime,Infrastructure"
    }

    livekit = container "LiveKit" {
        description "Публичная точка входа для аудио, видео и демонстрации экрана во время собеседований."
        technology "LiveKit / WebRTC"
        tags "Media,Infrastructure"
    }
}

group "Событийная платформа" {
    kafka = container "Kafka" {
        description "Шина событий для асинхронного взаимодействия предметных областей."
        technology "Apache Kafka"
        tags "EventBus,Infrastructure"
    }

    schemaRegistry = container "Schema Registry" {
        description "Хранение, версионирование и проверка совместимости схем Avro."
        technology "Schema Registry / Avro"
        tags "EventInfrastructure,Infrastructure"
    }

    debezium = container "Debezium" {
        description "Публикует доменные события из транзакционных outbox в Kafka."
        technology "Debezium"
        tags "EventInfrastructure,Infrastructure"
    }
}
