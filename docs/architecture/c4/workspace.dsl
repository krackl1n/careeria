workspace "Careeria" "Архитектурная модель рекрутинговой системы Careeria" {

    !identifiers hierarchical

    model {

        candidate = person "Кандидат" {
            description "Ищет работу, откликается на вакансии и проходит процесс найма."
        }

        recruiter = person "Рекрутер" {
            description "Публикует вакансии и ведёт кандидатов по процессу найма."
        }

        hiringManager = person "Нанимающий руководитель" {
            description "Участвует в собеседованиях, оценке кандидатов и принятии решений о найме."
        }

        companyAdmin = person "Администратор компании" {
            description "Управляет компанией, пользователями и ролями."
        }

        identityProvider = softwareSystem "Identity Provider" {
            description "Внешний провайдер аутентификации по OAuth 2.0 и OpenID Connect."
            tags "ExternalSystem"
        }

        notificationProviders = softwareSystem "Notification Providers" {
            description "Внешние сервисы доставки электронной почты, push- и SMS-уведомлений."
            tags "ExternalSystem"
        }

        employerHrSystem = softwareSystem "Employer HR System" {
            description "Внешняя корпоративная система работодателя, которая обменивается с Careeria данными о кандидатах и результатах найма."
            tags "ExternalSystem"
        }

        platform = softwareSystem "Careeria" {
            description "Рекрутинговая система полного цикла: от публикации вакансии и отклика до оценки, собеседования и найма кандидата."

            group "Web" {
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
            }

            apiGateway = container "API Gateway" {
                description "Единая точка входа во внутренние сервисы: маршрутизация, ограничение запросов, трассировка и передача доверенного контекста пользователя."
                technology "HTTP / gRPC"
                tags "Gateway"
            }

            group "Candidate Context" {
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

            group "Company Context" {
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

            group "Vacancy Context" {
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

            group "Hiring Context" {
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

            group "Assessment Context" {
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

            group "Communication Context" {
                communicationService = container "Communication Service" {
                    description "Диалоги, участники, сообщения, вложения, реакции и отметки о прочтении."
                    tags "BusinessService"
                }

                communicationDb = container "Communication DB" {
                    description "Данные Communication Service и его транзакционный outbox."
                    technology "PostgreSQL"
                    tags "Database"
                }
            }

            group "Identity and Access" {
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

            group "Files" {
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

            group "Event Platform" {
                kafka = container "Kafka" {
                    description "Шина событий для асинхронного взаимодействия предметных областей."
                    technology "Apache Kafka"
                    tags "EventBus"
                }

                schemaRegistry = container "Schema Registry" {
                    description "Хранение, версионирование и проверка совместимости схем Avro."
                    technology "Schema Registry / Avro"
                    tags "EventInfrastructure"
                }

                debezium = container "Debezium" {
                    description "Публикует доменные события из транзакционных outbox в Kafka."
                    technology "Debezium"
                    tags "EventInfrastructure"
                }
            }

            group "Realtime and Media" {
                realtimePublisher = container "Realtime Publisher" {
                    description "Получает события из Kafka, преобразует их для доставки в реальном времени и публикует в Centrifugo."
                    tags "PlatformService"
                }

                centrifugo = container "Centrifugo" {
                    description "Точка входа и WebSocket-транспорт для доставки событий в реальном времени."
                    technology "Centrifugo / WebSocket"
                    tags "Realtime"
                }

                livekit = container "LiveKit" {
                    description "Передаёт аудио, видео и демонстрацию экрана во время собеседований."
                    technology "LiveKit / WebRTC"
                    tags "Media"
                }
            }

            group "Platform Services" {
                notificationService = container "Notification Service" {
                    description "Формирует и доставляет уведомления в приложении, по электронной почте, через push и SMS."
                    tags "PlatformService"
                }

                integrationService = container "Integration Service" {
                    description "Принимает и отправляет webhooks, преобразует внешние сообщения в события Careeria и управляет надёжной доставкой."
                    tags "PlatformService"
                }
            }
        }

        candidate -> platform "Создаёт профиль и резюме, ищет вакансии, откликается и проходит отбор"
        recruiter -> platform "Публикует вакансии, ведёт кандидатов и организует собеседования"
        hiringManager -> platform "Проводит собеседования, оценивает кандидатов и принимает решения"
        companyAdmin -> platform "Управляет профилем компании, сотрудниками и ролями"

        platform -> identityProvider "Использует внешнюю аутентификацию" "OAuth 2.0 / OpenID Connect"
        platform -> notificationProviders "Отправляет уведомления"
        platform -> employerHrSystem "Передаёт сведения о ходе и результатах найма" "REST API / Webhooks"
        employerHrSystem -> platform "Передаёт изменения данных кандидатов и сотрудников" "Webhooks"

        candidate -> platform.web "Использует" "HTTPS"
        recruiter -> platform.web "Использует" "HTTPS"
        hiringManager -> platform.web "Использует" "HTTPS"
        companyAdmin -> platform.web "Использует" "HTTPS"

        platform.web -> platform.bff "Вызывает API веб-приложения" "HTTPS/JSON"
        platform.bff -> platform.apiGateway "Вызывает внутренний API" "HTTPS/gRPC"

        platform.apiGateway -> platform.identityService "Проверяет аутентификацию и получает данные пользователя" "gRPC"
        platform.identityService -> identityProvider "Выполняет внешнюю аутентификацию" "OAuth 2.0 / OpenID Connect"

        platform.apiGateway -> platform.candidateService "Передаёт запросы с доверенным контекстом пользователя" "gRPC"
        platform.apiGateway -> platform.companyService "Передаёт запросы с доверенным контекстом пользователя" "gRPC"
        platform.apiGateway -> platform.vacancyService "Передаёт запросы с доверенным контекстом пользователя" "gRPC"
        platform.apiGateway -> platform.hiringService "Передаёт запросы с доверенным контекстом пользователя" "gRPC"
        platform.apiGateway -> platform.assessmentService "Передаёт запросы с доверенным контекстом пользователя" "gRPC"
        platform.apiGateway -> platform.communicationService "Передаёт запросы с доверенным контекстом пользователя" "gRPC"
        platform.apiGateway -> platform.notificationService "Передаёт запросы с доверенным контекстом пользователя" "gRPC"
        platform.apiGateway -> platform.fileService "Передаёт запросы с доверенным контекстом пользователя" "gRPC"
        platform.apiGateway -> platform.integrationService "Передаёт входящие webhooks" "HTTP/gRPC"

        employerHrSystem -> platform.apiGateway "Отправляет события интеграции" "HTTPS/Webhooks"
        platform.integrationService -> employerHrSystem "Отправляет данные и события найма" "REST API / Webhooks"

        platform.candidateService -> platform.openFga "Проверяет права доступа" "OpenFGA API"
        platform.companyService -> platform.openFga "Проверяет права доступа" "OpenFGA API"
        platform.vacancyService -> platform.openFga "Проверяет права доступа" "OpenFGA API"
        platform.hiringService -> platform.openFga "Проверяет права доступа" "OpenFGA API"
        platform.assessmentService -> platform.openFga "Проверяет права доступа" "OpenFGA API"
        platform.communicationService -> platform.openFga "Проверяет права доступа" "OpenFGA API"
        platform.fileService -> platform.openFga "Проверяет права доступа" "OpenFGA API"

        platform.candidateService -> platform.candidateDb "Хранит данные кандидатов и исходящие события" "SQL"
        platform.companyService -> platform.companyDb "Хранит данные компаний и исходящие события" "SQL"
        platform.vacancyService -> platform.vacancyDb "Хранит данные вакансий и исходящие события" "SQL"
        platform.hiringService -> platform.hiringDb "Хранит данные процесса найма и исходящие события" "SQL"
        platform.assessmentService -> platform.assessmentDb "Хранит данные оценки и исходящие события" "SQL"
        platform.communicationService -> platform.communicationDb "Хранит переписку и исходящие события" "SQL"
        platform.identityService -> platform.identityDb "Хранит учётные данные, сессии и исходящие события" "SQL"

        platform.fileService -> platform.objectStorage "Сохраняет и получает файлы" "S3 API"

        platform.candidateDb -> platform.debezium "Передаёт записи журнала исходящих сообщений" "PostgreSQL WAL" {
            tags "CDC"
        }

        platform.companyDb -> platform.debezium "Передаёт записи журнала исходящих сообщений" "PostgreSQL WAL" {
            tags "CDC"
        }

        platform.vacancyDb -> platform.debezium "Передаёт записи журнала исходящих сообщений" "PostgreSQL WAL" {
            tags "CDC"
        }

        platform.hiringDb -> platform.debezium "Передаёт записи журнала исходящих сообщений" "PostgreSQL WAL" {
            tags "CDC"
        }

        platform.assessmentDb -> platform.debezium "Передаёт записи журнала исходящих сообщений" "PostgreSQL WAL" {
            tags "CDC"
        }

        platform.communicationDb -> platform.debezium "Передаёт записи журнала исходящих сообщений" "PostgreSQL WAL" {
            tags "CDC"
        }

        platform.identityDb -> platform.debezium "Передаёт записи журнала исходящих сообщений" "PostgreSQL WAL" {
            tags "CDC"
        }

        platform.debezium -> platform.kafka "Публикует доменные события" "Avro" {
            tags "Async"
        }

        platform.debezium -> platform.schemaRegistry "Регистрирует и получает схемы событий" "Avro"

        platform.kafka -> platform.hiringService "Доставляет события, влияющие на процесс найма" "Avro" {
            tags "Async"
        }

        platform.kafka -> platform.notificationService "Доставляет события для формирования уведомлений" "Avro" {
            tags "Async"
        }

        platform.kafka -> platform.realtimePublisher "Доставляет события для отправки в реальном времени" "Avro" {
            tags "RealtimeFlow"
        }

        platform.kafka -> platform.authzProjection "Доставляет события об изменениях ролей и отношений доступа" "Avro" {
            tags "Async"
        }

        platform.kafka -> platform.integrationService "Доставляет события для внешних систем" "Avro" {
            tags "Async"
        }

        platform.integrationService -> platform.kafka "Публикует события, полученные из внешних систем" "Avro" {
            tags "Async"
        }

        platform.hiringService -> platform.schemaRegistry "Получает схемы обрабатываемых событий" "Avro"
        platform.notificationService -> platform.schemaRegistry "Получает схемы обрабатываемых событий" "Avro"
        platform.realtimePublisher -> platform.schemaRegistry "Получает схемы обрабатываемых событий" "Avro"
        platform.authzProjection -> platform.schemaRegistry "Получает схемы обрабатываемых событий" "Avro"
        platform.integrationService -> platform.schemaRegistry "Регистрирует и получает схемы интеграционных событий" "Avro"

        platform.authzProjection -> platform.openFga "Изменяет роли и отношения доступа" "OpenFGA API"

        platform.realtimePublisher -> platform.centrifugo "Публикует сообщения для доставки в реальном времени" "Centrifugo API" {
            tags "RealtimeFlow"
        }

        platform.web -> platform.centrifugo "Устанавливает соединение для событий в реальном времени" "WebSocket" {
            tags "RealtimeFlow"
        }

        platform.centrifugo -> platform.web "Доставляет события пользователю" "WebSocket" {
            tags "RealtimeFlow"
        }

        platform.hiringService -> platform.livekit "Создаёт комнаты и выдаёт доступ к медиасессиям" "LiveKit API" {
            tags "MediaFlow"
        }

        platform.web -> platform.livekit "Передаёт аудио, видео и демонстрацию экрана" "WebRTC" {
            tags "MediaFlow"
        }

        platform.notificationService -> notificationProviders "Отправляет письма, push- и SMS-уведомления"
    }

    views {

        systemContext platform "C4-L1-SystemContext" {
            title "Careeria — системный контекст (C4, уровень 1)"
            include *
            autoLayout lr 400 300
        }

        container platform "C4-L2-Container" {
            title "Careeria — контейнеры (C4, уровень 2)"
            include *
            autoLayout lr 500 300
        }

        styles {

            element "Element" {
                shape RoundedBox
                background #64748b
                color #ffffff
                stroke #475569
            }

            element "Person" {
                shape Person
                background #334155
                color #ffffff
            }

            element "Software System" {
                background #2563eb
                color #ffffff
            }

            element "ExternalSystem" {
                background #64748b
                color #ffffff
                border dashed
            }

            element "Frontend" {
                shape WebBrowser
                background #2563eb
                color #ffffff
            }

            element "Edge" {
                background #0ea5e9
                color #ffffff
            }

            element "Gateway" {
                background #0284c7
                color #ffffff
                strokeWidth 4
            }

            element "BusinessService" {
                background #16a34a
                color #ffffff
            }

            element "PlatformService" {
                background #0891b2
                color #ffffff
            }

            element "Authorization" {
                background #7c3aed
                color #ffffff
            }

            element "EventBus" {
                shape Pipe
                background #f59e0b
                color #111827
                stroke #b45309
                strokeWidth 4
            }

            element "EventInfrastructure" {
                background #d97706
                color #ffffff
            }

            element "Realtime" {
                shape Hexagon
                background #db2777
                color #ffffff
                stroke #9d174d
                strokeWidth 4
            }

            element "Media" {
                shape Hexagon
                background #9333ea
                color #ffffff
                stroke #6b21a8
                strokeWidth 4
            }

            element "Database" {
                shape Cylinder
                background #475569
                color #ffffff
            }

            element "ObjectStorage" {
                shape Bucket
                background #475569
                color #ffffff
            }

            relationship "Relationship" {
                color #64748b
                thickness 2
                routing Orthogonal
            }

            relationship "Async" {
                color #d97706
                style dashed
                thickness 2
            }

            relationship "RealtimeFlow" {
                color #db2777
                thickness 3
            }

            relationship "MediaFlow" {
                color #9333ea
                thickness 3
            }

            relationship "CDC" {
                color #a16207
                style dotted
                thickness 2
            }
        }
    }
}
