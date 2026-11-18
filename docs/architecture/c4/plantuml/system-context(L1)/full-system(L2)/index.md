# Полная система (L2)

<!-- excerpt -->

Серверная часть приложения Next.js передаёт запросы в API Gateway. Шлюз вызывает доменные сервисы по gRPC, а изменения, на которые должны отреагировать другие сервисы, передаются через Kafka.

Каждый прикладной сервис отвечает за определённую область и самостоятельно управляет своими данными. Синхронный запрос проходит от веб-приложения через шлюз к API владельца ресурса. После сохранения изменения событие из outbox поступает через Debezium в Kafka, где его получают заинтересованные сервисы. Отдельные контейнеры обеспечивают проверку доступа, доставку обновлений и проведение медиасессий.

```plantuml
@startuml
!include architecture/c4/plantuml/system-context(L1)/full-system(L2)/full-system.puml
@enduml
```

<p align="center">Рисунок - Контейнеры платформы Careeria</p>

<p align="right">Таблица - Элементы. Контейнеры платформы Careeria</p>

| Элемент | Тип или технология | Назначение и ответственность |
|---|---|---|
| Кандидат | Пользователь | Ищет работу, откликается на вакансии и проходит процесс найма. |
| Рекрутер | Пользователь | Публикует вакансии и ведёт кандидатов по процессу найма. |
| Нанимающий руководитель | Пользователь | Участвует в собеседованиях, оценке кандидатов и принятии решений. |
| Администратор компании | Пользователь | Управляет компанией, пользователями и ролями. |
| Identity Provider | Система | Внешний провайдер удостоверения личности на базе Keycloak. Identity Service использует OAuth 2.0 и OIDC для федеративного входа и связывания внешней учётной записи с пользователем Careeria. |
| Notification Providers | Система | Внешние поставщики доставки сообщений. SendGrid обрабатывает email, Firebase Cloud Messaging доставляет push, Twilio отправляет SMS по запросам сервиса уведомлений. |
| Employer HR System | Система | Кадровая система работодателя. Принимает результаты найма из Careeria и передаёт изменения кадровых данных через согласованные вебхуки. |
| Web Application | Next.js, React, TypeScript, Tailwind CSS | Интерфейс React работает в браузере. Серверный BFF обрабатывает пользовательскую сессию и обращается к API Gateway. Страницы и BFF развёртываются вместе в одном приложении Next.js. |
| API Gateway | Go, grpc-gateway, grpc-go, OpenTelemetry | API Gateway принимает HTTP-запрос от BFF, проверяет предъявленный токен по открытым ключам Identity и применяет политику доступа к методу. Доменный сервис получает вызов по gRPC. |
| Candidate Service | Go, gRPC, Protobuf, pgx, sqlc | В Candidate Service один компонент ведёт профиль кандидата, другой управляет версиями резюме. Данные профиля и версий хранятся в Candidate DB, а загрузку и скачивание документов обеспечивает File Service. |
| Candidate DB | PostgreSQL | Профили, статусы, метаданные версий резюме, cleanup jobs и transactional outbox; без файлов. |
| Company Service | Go, gRPC, Protobuf, pgx, sqlc | Company Service управляет карточкой компании, её участниками и ролями. Перед изменением данных сервис проверяет полномочия через OpenFGA и сохраняет изменения вместе с событиями в Company DB. |
| Company DB | PostgreSQL | Схема Company Service и таблица transactional outbox. |
| Vacancy Service | Go, gRPC, Protobuf, pgx, sqlc | Vacancy Service хранит содержание и условия вакансии. При публикации сервис проверяет права рекрутера и готовность вакансии, затем фиксирует её новый статус в Vacancy DB. |
| Vacancy DB | PostgreSQL | Вакансии, навыки и transactional outbox. |
| Hiring Service | Go, gRPC, Protobuf, pgx, sqlc | Hiring Service ведёт отклик от первого этапа отбора до предложения о работе. Компоненты откликов, интервью и предложений согласуют переходы между этапами и сохраняют историю в Hiring DB. |
| Hiring DB | PostgreSQL | Applications, pipeline transitions, interviews, offers и transactional outbox. |
| Assessment Service | Go, gRPC, Protobuf, pgx, sqlc | Assessment Service управляет шаблонами заданий и попытками кандидатов. Для проверки кода он вызывает Code Execution Runner, затем рассчитывает балл и сохраняет результат в Assessment DB. |
| Code Execution Runner | gRPC, Docker/gVisor | Code Execution Runner запускает решение и тесты в изолированной среде с ограничениями времени и памяти. Отчёт получает Assessment Service, который рассчитывает итоговый балл. |
| Assessment DB | PostgreSQL | Assessments, sessions, submissions, evaluations и transactional outbox. |
| Communication Service | Go, gRPC, Protobuf, pgx, sqlc | Communication Service ведёт диалоги, сообщения и отметки о прочтении. Вложения обслуживает File Service, а события о новых сообщениях поступают подписчикам через Kafka. |
| Communication DB | PostgreSQL | Conversations, messages, read receipts и transactional outbox. |
| Notification Service | Go, Kafka consumer, REST clients | Notification Service принимает команды и события из Kafka, выбирает шаблон и создаёт задачу доставки. Обработчики сохраняют уведомления в личном кабинете или передают их провайдерам email, push и SMS. |
| Notification DB | PostgreSQL | Шаблоны уведомлений, delivery jobs и in-app Inbox. |
| Identity Service | Go, gRPC, OAuth 2.0/OIDC, JWT, Argon2id | Identity Service регистрирует пользователей, проверяет учётные данные, выпускает токены и управляет сессиями. Для входа через внешнего провайдера сервис поддерживает OIDC. |
| Identity DB | PostgreSQL | Пользователи, credentials, refresh sessions и transactional outbox. |
| AuthZ Synchronization Service | Go, Kafka consumer, OpenFGA SDK | AuthZ Synchronization Service читает доменные события из Kafka и обновляет отношения доступа в OpenFGA. Например, после публикации вакансии сервис добавляет право на чтение её карточки. |
| OpenFGA | OpenFGA, PostgreSQL | Сервис проверки доступа на основе отношений между субъектом и ресурсом. Доменные сервисы запрашивают решение о доступе, а обработчик синхронизации обновляет отношения по событиям платформы. |
| File Service | Go, gRPC, MinIO Go SDK | File Service выдаёт временную ссылку на загрузку в объектное хранилище, проверяет загруженную версию и связывает её с ресурсом платформы. Привязка определяет, какую версию файла можно скачать. |
| File DB | PostgreSQL | Метаданные файлов, сеансы загрузки, привязки и задачи очистки. |
| Object Storage | MinIO, S3 API | Приватное объектное хранилище файлов на базе MinIO. File Service управляет версиями и ограниченными ссылками, а браузер передаёт содержимое по выданному разрешению. |
| Integration Service | Go, HTTP, gRPC, Kafka, pgx, sqlc | Integration Service проверяет подпись входящего вебхука и передаёт принятое событие в Kafka. Для исходящих вебхуков он ведёт подписки работодателей и повторяет доставку при временном сбое. |
| Integration DB | PostgreSQL | Подключения, подписки, inbox, delivery jobs, попытки и transactional outbox. |
| Realtime Publisher | Go, Kafka consumer, Centrifugo HTTP API | Realtime Publisher читает события из Kafka, определяет канал получателя и публикует сообщение в Centrifugo. Centrifugo доставляет его в браузер по WebSocket. |
| Centrifugo | Centrifugo, WebSocket, JWT | Сервис подключений браузеров и каналов обновления интерфейса. Принимает публикации Realtime Publisher и доставляет сообщения авторизованным подписчикам по WebSocket. |
| LiveKit | LiveKit, WebRTC, LiveKit React SDK | Медиасервер для интервью и Live Coding. Сервисы управляют комнатами через API, а браузеры участников передают аудио, видео и демонстрацию экрана через WebRTC. |
| Kafka | Apache Kafka, KRaft | Событийная шина платформы. Хранит доменные сообщения в топиках и передаёт их группам потребителей для независимой асинхронной обработки. |
| Schema Registry | Apicurio Registry, Avro | Хранилище версий схем событий Avro на базе Apicurio Registry. Издатели и потребители получают схемы для сериализации и чтения доменных сообщений. |
| Debezium | Debezium PostgreSQL Connector | Компонент захвата зафиксированных изменений PostgreSQL. Читает transactional outbox через журнал WAL и публикует доменные события в Kafka. |

<p align="right">Таблица - Связи. Контейнеры платформы Careeria</p>

| Источник | Получатель | Содержание взаимодействия | Протокол или способ вызова |
|---|---|---|---|
| Кандидат | Web Application | Использует. | HTTPS |
| Рекрутер | Web Application | Использует. | HTTPS |
| Нанимающий руководитель | Web Application | Использует. | HTTPS |
| Администратор компании | Web Application | Использует. | HTTPS |
| Web Application | API Gateway | Серверный BFF вызывает API. | HTTP / JSON |
| API Gateway | Identity Service | Проверяет пользователя. Запрос и ответ передаются как сообщения Protobuf. | gRPC |
| Identity Service | Identity Provider | Аутентифицирует. | OAuth 2.0 / OIDC |
| API Gateway | Candidate Service | Получает профили. Запрос и ответ передаются как сообщения Protobuf. | gRPC |
| API Gateway | Company Service | Управляет компаниями. Запрос и ответ передаются как сообщения Protobuf. | gRPC |
| API Gateway | Vacancy Service | Управляет вакансиями. Запрос и ответ передаются как сообщения Protobuf. | gRPC |
| API Gateway | Hiring Service | Ведёт процесс найма. Запрос и ответ передаются как сообщения Protobuf. | gRPC |
| API Gateway | Assessment Service | Управляет оцениванием. Запрос и ответ передаются как сообщения Protobuf. | gRPC |
| API Gateway | Communication Service | Обменивается сообщениями. Запрос и ответ передаются как сообщения Protobuf. | gRPC |
| API Gateway | Notification Service | Управляет уведомлениями. Запрос и ответ передаются как сообщения Protobuf. | gRPC |
| API Gateway | File Service | Работает с файлами. Запрос и ответ передаются как сообщения Protobuf. | gRPC |
| API Gateway | Integration Service | Вебхуки и управление. Запрос и ответ передаются как сообщения Protobuf. | HTTP / gRPC |
| Employer HR System | API Gateway | Отправляет webhook. | HTTPS / JSON |
| Integration Service | Employer HR System | Доставляет исходящие webhooks. | HTTPS / JSON |
| Candidate Service | OpenFGA | Проверяет доступ. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Company Service | OpenFGA | Проверяет доступ. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Vacancy Service | OpenFGA | Проверяет доступ. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Hiring Service | OpenFGA | Проверяет доступ. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Assessment Service | OpenFGA | Проверяет доступ. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Communication Service | OpenFGA | Проверяет доступ. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| File Service | OpenFGA | Проверяет доступ. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Integration Service | OpenFGA | Проверяет управление интеграциями. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Candidate Service | Candidate DB | Хранит профили и метаданные резюме. Получатель выполняет чтение или запись в базе данных сервиса. | SQL |
| Candidate Service | File Service | Загружает и связывает резюме. Запрос и ответ передаются как сообщения Protobuf. | gRPC |
| Company Service | Company DB | Хранит компании и роли. Получатель выполняет чтение или запись в базе данных сервиса. | SQL |
| Vacancy Service | Vacancy DB | Хранит вакансии. Получатель выполняет чтение или запись в базе данных сервиса. | SQL |
| Hiring Service | Hiring DB | Хранит отклики и этапы. Получатель выполняет чтение или запись в базе данных сервиса. | SQL |
| Assessment Service | Code Execution Runner | Передаёт решение на проверку. Запрос и ответ передаются как сообщения Protobuf. | gRPC |
| Assessment Service | Assessment DB | Хранит задания и оценки. Получатель выполняет чтение или запись в базе данных сервиса. | SQL |
| Communication Service | Communication DB | Хранит сообщения. Получатель выполняет чтение или запись в базе данных сервиса. | SQL |
| Notification Service | Notification DB | Хранит шаблоны, jobs и Inbox. Получатель выполняет чтение или запись в базе данных сервиса. | SQL |
| Identity Service | Identity DB | Хранит учётные данные. Получатель выполняет чтение или запись в базе данных сервиса. | SQL |
| Integration Service | Integration DB | Хранит inbox, настройки и доставки. Получатель выполняет чтение или запись в базе данных сервиса. | SQL |
| File Service | File DB | Сохраняет метаданные и привязки. Получатель выполняет чтение или запись в базе данных сервиса. | SQL |
| File Service | Object Storage | Читает и записывает. Выполняется операция с содержимым или версией объекта в хранилище. | S3 API |
| Web Application | Object Storage | Загружает файл по presigned URL. Выполняется операция с содержимым или версией объекта в хранилище. | HTTPS / S3 API |
| Candidate DB | Debezium | Передаёт outbox. Коннектор читает зафиксированные изменения из журнала PostgreSQL. | PostgreSQL WAL |
| Company DB | Debezium | Передаёт outbox. Коннектор читает зафиксированные изменения из журнала PostgreSQL. | PostgreSQL WAL |
| Vacancy DB | Debezium | Передаёт outbox. Коннектор читает зафиксированные изменения из журнала PostgreSQL. | PostgreSQL WAL |
| Hiring DB | Debezium | Передаёт outbox. Коннектор читает зафиксированные изменения из журнала PostgreSQL. | PostgreSQL WAL |
| Assessment DB | Debezium | Передаёт outbox. Коннектор читает зафиксированные изменения из журнала PostgreSQL. | PostgreSQL WAL |
| Communication DB | Debezium | Передаёт outbox. Коннектор читает зафиксированные изменения из журнала PostgreSQL. | PostgreSQL WAL |
| Identity DB | Debezium | Передаёт outbox. Коннектор читает зафиксированные изменения из журнала PostgreSQL. | PostgreSQL WAL |
| Integration DB | Debezium | Передаёт outbox. Коннектор читает зафиксированные изменения из журнала PostgreSQL. | PostgreSQL WAL |
| Debezium | Schema Registry | Получает схемы. | Registry API |
| Hiring Service | Schema Registry | Получает схемы. | Registry API |
| Notification Service | Schema Registry | Получает схемы. | Registry API |
| Realtime Publisher | Schema Registry | Получает схемы. | Registry API |
| AuthZ Synchronization Service | Schema Registry | Получает схемы. | Registry API |
| Integration Service | Schema Registry | Получает схемы событий. | Registry API |
| Debezium | Kafka | Публикует события. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro |
| Kafka | Hiring Service | Доставляет события найма. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro |
| Kafka | Notification Service | Доставляет события уведомлений. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro |
| Kafka | Realtime Publisher | Доставляет realtime-события. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro |
| Kafka | AuthZ Synchronization Service | Доставляет изменения ролей. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro |
| Kafka | Integration Service | Доставляет события интеграций. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro |
| AuthZ Synchronization Service | OpenFGA | Обновляет роли и связи. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Realtime Publisher | Centrifugo | Публикует сообщения. | Centrifugo API |
| Web Application | Centrifugo | Подключается. Данные передаются по постоянному соединению с браузером. | WebSocket |
| Centrifugo | Web Application | Доставляет сообщения. Данные передаются по постоянному соединению с браузером. | WebSocket |
| Hiring Service | LiveKit | Создаёт комнаты. | LiveKit API |
| Web Application | LiveKit | Передаёт медиапоток. По соединению передаются медиапотоки участников. | WebRTC |
| Notification Service | Notification Providers | Отправляет уведомления. | Email / Push / SMS |

<!-- /excerpt -->

## Путь пользовательского запроса

`Browser → Web Application (Next.js с BFF) → API Gateway → доменный сервис → PostgreSQL`.

Каждый сервис владеет своими данными. Синхронные обращения идут через API владельца; событийные взаимодействия используют Kafka.

## Детальные представления

- [Коммуникации](../../communication%28L2%29/index.md).
- [Событийная архитектура](../../event-driven%28L2%29/index.md).

## Компоненты контейнеров (L3) {#components}

Компонентные диаграммы раскрывают внутреннюю структуру отдельного контейнера L2: адаптеры, прикладные сервисы, интеграции и слой хранения. Внешние контейнеры на таких диаграммах остаются зависимостями и не раскрываются.

<p align="right">Таблица - Компоненты контейнеров (L3) (Полная система (L2))</p>

| Диаграмма | Компоненты внутри контейнера | Область сценариев | Что раскрывает страница |
|---|---|---|---|
| [Company Service](company-service%28L3%29/index.md) | 7 | 12 сценариев, названных на L3 | Управление компанией и верификацией, членство и роли, ReBAC-проверки, транзакции и синхронизация доступа. |
| [Notification Service](notification-service%28L3%29/index.md) | 10 | 4 именованных прикладных сценария и 3 входных/пользовательских сценария | Создание и отмена jobs, конкурентный захват, исполнение и повторы, Kafka, постоянный Inbox и in-app stream. |
| [Hiring Service](hiring-service%28L3%29/index.md) | 13 | 18 сценариев; 7 RPC приведены в исходной спецификации | Отклики и FSM, интервью и feedback, жизненный цикл оффера, конференции, внешняя синхронизация и межагрегатные транзакции. |
| [Vacancy Service](vacancy-service%28L3%29/index.md) | 6 | 8 сценариев; 6 RPC перечислены в исходной спецификации | Содержание вакансии, требования и компенсация, жизненный цикл доменного агрегата Vacancy, видимость и выборки, авторизация и outbox. |
| [Candidate Service](candidate-service%28L3%29/index.md) | 9 | 24 группы сценариев; предлагаемые контракты | Профиль и профессиональные данные, статус поиска и видимость, версии резюме в S3 через File Service, безопасное скачивание, cleanup и снимки для найма. |
| [Integration Service](integration-service%28L3%29/index.md) | 13 | 21 группа сценариев; предлагаемые контракты | Входящие и исходящие webhooks, подключения компаний, подписи и ключи, inbox/outbox, подписки, deliveries, leases, retry/redrive и SSRF-защита. |
| [Identity Service](identity-service%28L3%29/index.md) | 12 | 20 сценариев | Регистрация, credentials, access/refresh токены , ротация сессий, обнаружение reuse, Debezium CDC и адаптер для внешних интеграций. |
| [Assessment Service](assessment-service%28L3%29/index.md) | 12 | 14 сценариев | Шаблоны тестов и заданий, Live Coding через LiveKit, паттерн Strategy для скоринга, раннер в Sandbox, сессии FSM и Debezium outbox. |
| [Communication Service](communication-service%28L3%29/index.md) | 10 | 17 сценариев | Диалоги и треды, ReBAC-доступ через OpenFGA, вложения через File Service в S3, outbox для Centrifugo и Notification Service, soft delete и watermarks. |
| [File Service](file-service%28L3%29/index.md) | 7 | 10 сценариев файловых операций | Загрузка, проверка версии, привязки и очистка. |
| [API Gateway](api-gateway%28L3%29/index.md) | 5 | Входящие HTTP-запросы | Маршрутизация, проверка JWT, политика RPC и преобразование ответа. |
| [Web Application](web-application%28L3%29/index.md) | 5 | Страницы и клиентские соединения | SSR, интерфейс React, сессия и серверный BFF в одном приложении Next.js. |
| [Realtime Publisher](realtime-publisher%28L3%29/index.md) | 3 | Доставка доменных событий | Отбор событий, выбор канала и публикация в Centrifugo. |
| [AuthZ Synchronization Service](authz-synchronization-service%28L3%29/index.md) | 3 | Проекция отношений доступа | Преобразование доменных событий в изменения отношений OpenFGA. |
| [Code Execution Runner](code-execution-runner%28L3%29/index.md) | 4 | Исполнение практических заданий | Изоляция запуска, контроль ресурсов и отчёт для Assessment Service. |

В таблице указаны компоненты, входящие в границу каждого контейнера. На схемах также показаны соседние контейнеры и протоколы обмена.

Диаграммы описывают целевую архитектуру платформы. Страницы доменных сервисов также раскрывают сценарии, границы транзакций и вопросы проектирования.
