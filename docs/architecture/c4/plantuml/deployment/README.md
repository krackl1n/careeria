# Развёртывание в Kubernetes

Целевая схема production-развёртывания Careeria. Приложения и хранилища работают **внутри Kubernetes**; PostgreSQL, Kafka, Redis и MinIO используют постоянные тома. Это архитектурный проект: Kubernetes-манифестов и Helm chart в репозитории пока нет.

Число узлов, реплик и пороги ниже — стартовые параметры для проектирования. Они не являются результатом расчёта производительности и уточняются нагрузочными испытаниями. Облачный провайдер, версия Kubernetes, реализация Ingress controller, CSI и конкретные операторы пока не выбраны.

## Размещение компонентов

<!-- excerpt -->

В кластере Kubernetes Ingress направляет запросы к приложению Next.js. Веб-приложение, API Gateway и доменные сервисы масштабируются по отдельности. Серверный BFF входит в развёртывание Next.js.

Запрос пользователя проходит через публичный балансировщик и Ingress к Next.js, после чего BFF вызывает шлюз и нужный API. Приложения и платформенные обработчики размещаются на узлах приложений, медиасессии обслуживаются отдельным пулом. Хранилища используют узлы данных и постоянные тома. Вложенность показывает размещение исполняемых экземпляров, подписи кратности задают число реплик.

```plantuml
@startuml
!include architecture/c4/plantuml/deployment/kubernetes-production.puml
@enduml
```

<p align="center">Рисунок - Размещение компонентов</p>

<p align="right">Таблица - Элементы. Размещение компонентов</p>

| Элемент | Тип или технология | Назначение и ответственность |
|---|---|---|
| Устройство пользователя | Workstation / Smart Device | Размещает вложенные экземпляры компонентов в окружении Workstation / Smart Device. |
| Браузер пользователя | Web browser | Загружает интерфейс Careeria из кластера и выполняет клиентский JavaScript. |
| Публичный балансировщик | L4 / TCP | Передаёт HTTPS и WSS на Ingress; порт 443. |
| Инфраструктура Careeria | Production Kubernetes Cluster | Размещает вложенные экземпляры компонентов в окружении Production Kubernetes Cluster. |
| Пул узлов приложений × 3..N | Linux / container runtime | Размещает вложенные экземпляры компонентов в окружении Linux / container runtime. |
| Ingress Controller Pods × 2 | Kubernetes Pod / Nginx или Envoy | Размещает вложенные экземпляры компонентов в окружении Kubernetes Pod / Nginx или Envoy. |
| Ingress Controller | HTTP / WebSocket Reverse Proxy | TLS-терминация, балансировка и маршрутизация; порт 443. |
| Web Application Pods × 2..10 | Kubernetes Pod / Node.js | Размещает вложенные экземпляры компонентов в окружении Kubernetes Pod / Node.js. |
| Web Application | Next.js / React / TypeScript | Интерфейс React работает в браузере. Серверный BFF обрабатывает пользовательскую сессию и обращается к API Gateway. Страницы и BFF развёртываются вместе в одном приложении Next.js. |
| API Gateway | Go / grpc-gateway · Pods × 2..10 | API Gateway принимает HTTP-запрос от BFF, проверяет предъявленный токен по открытым ключам Identity и применяет политику доступа к методу. Доменный сервис получает вызов по gRPC. |
| Candidate Service | Go / gRPC · Pods × 2..10 | В Candidate Service один компонент ведёт профиль кандидата, другой управляет версиями резюме. Данные профиля и версий хранятся в Candidate DB, а загрузку и скачивание документов обеспечивает File Service. |
| Company Service | Go / gRPC · Pods × 2..10 | Company Service управляет карточкой компании, её участниками и ролями. Перед изменением данных сервис проверяет полномочия через OpenFGA и сохраняет изменения вместе с событиями в Company DB. |
| Vacancy Service | Go / gRPC · Pods × 2..10 | Vacancy Service хранит содержание и условия вакансии. При публикации сервис проверяет права рекрутера и готовность вакансии, затем фиксирует её новый статус в Vacancy DB. |
| Hiring Service | Go / gRPC · Pods × 2..10 | Hiring Service ведёт отклик от первого этапа отбора до предложения о работе. Компоненты откликов, интервью и предложений согласуют переходы между этапами и сохраняют историю в Hiring DB. |
| Assessment Service | Go / gRPC · Pods × 2..10 | Assessment Service управляет шаблонами заданий и попытками кандидатов. Для проверки кода он вызывает Code Execution Runner, затем рассчитывает балл и сохраняет результат в Assessment DB. |
| Communication Service | Go / gRPC · Pods × 2..10 | Communication Service ведёт диалоги, сообщения и отметки о прочтении. Вложения обслуживает File Service, а события о новых сообщениях поступают подписчикам через Kafka. |
| Identity Service | Go / gRPC · Pods × 2..10 | Identity Service регистрирует пользователей, проверяет учётные данные, выпускает токены и управляет сессиями. Для входа через внешнего провайдера сервис поддерживает OIDC. |
| Integration Service | Go / gRPC · Pods × 2..10 | Integration Service проверяет подпись входящего вебхука и передаёт принятое событие в Kafka. Для исходящих вебхуков он ведёт подписки работодателей и повторяет доставку при временном сбое. |
| File Service | Go / gRPC · Pods × 2..10 | File Service выдаёт временную ссылку на загрузку в объектное хранилище, проверяет загруженную версию и связывает её с ресурсом платформы. Привязка определяет, какую версию файла можно скачать. |
| Notification Service | Go / gRPC · Pods × 2..10 | Notification Service принимает команды и события из Kafka, выбирает шаблон и создаёт задачу доставки. Обработчики сохраняют уведомления в личном кабинете или передают их провайдерам email, push и SMS. |
| Realtime Publisher | Go / Kafka consumer · Pods × 2..8 | Realtime Publisher читает события из Kafka, определяет канал получателя и публикует сообщение в Centrifugo. Centrifugo доставляет его в браузер по WebSocket. |
| AuthZ Synchronization Service | Go / OpenFGA SDK · Pods × 2..8 | AuthZ Synchronization Service читает доменные события из Kafka и обновляет отношения доступа в OpenFGA. Например, после публикации вакансии сервис добавляет право на чтение её карточки. |
| Centrifugo | Centrifugo / WebSocket · Pods × 2..N | Сервис подключений браузеров и каналов обновления интерфейса. Принимает публикации Realtime Publisher и доставляет сообщения авторизованным подписчикам по WebSocket. |
| OpenFGA | OpenFGA Engine · Pods × 2 | Сервис проверки доступа на основе отношений между субъектом и ресурсом. Доменные сервисы запрашивают решение о доступе, а обработчик синхронизации обновляет отношения по событиям платформы. |
| Медиа-узлы × 2..N | Linux / host networking | Размещает вложенные экземпляры компонентов в окружении Linux / host networking. |
| LiveKit Pods × 2..N | Kubernetes Pod / один Pod на узел | Размещает вложенные экземпляры компонентов в окружении Kubernetes Pod / один Pod на узел. |
| LiveKit | LiveKit / WebRTC SFU | Медиасервер для интервью и Live Coding. Сервисы управляют комнатами через API, а браузеры участников передают аудио, видео и демонстрацию экрана через WebRTC. |
| Узлы данных × 3..N | Linux / persistent volumes | Размещает вложенные экземпляры компонентов в окружении Linux / persistent volumes. |
| Общий PostgreSQL-кластер | Kubernetes StatefulSet / primary + 2 replicas / PVC | Размещает вложенные экземпляры компонентов в окружении Kubernetes StatefulSet / primary + 2 replicas / PVC. |
| PostgreSQL | PostgreSQL | Общая реляционная СУБД платформы. Сервисы используют собственные логические базы или схемы и роли подключения, сохраняя данные через SQL. Постоянные тома обеспечивают сохранение данных экземпляров кластера. |
| Kafka Pods × 3 | Kubernetes Pod / StatefulSet / PVC | Размещает вложенные экземпляры компонентов в окружении Kubernetes Pod / StatefulSet / PVC на экземпляр. |
| Kafka | Apache Kafka / KRaft | Событийная шина платформы. Хранит доменные сообщения в топиках и передаёт их группам потребителей для независимой асинхронной обработки. |

<p align="right">Таблица - Связи. Размещение компонентов</p>

| Источник | Получатель | Содержание взаимодействия | Протокол или способ вызова |
|---|---|---|---|
| Браузер пользователя | Публичный балансировщик | Загружает интерфейс и вызывает API. Данные передаются по постоянному соединению с браузером. | HTTPS / WSS, порт 443 |
| Публичный балансировщик | Ingress Controller | Передаёт TLS-соединения. | TCP, порт 443 |
| Ingress Controller | Web Application | Маршрутизирует страницы, ресурсы и API BFF. | HTTP |
| Web Application | API Gateway | Серверный BFF вызывает API. | HTTP / JSON |
| API Gateway | Candidate Service | Профили кандидатов. Запрос и ответ передаются как сообщения Protobuf. | gRPC / HTTP2 |
| API Gateway | Company Service | Компании и корпоративные роли. Запрос и ответ передаются как сообщения Protobuf. | gRPC / HTTP2 |
| API Gateway | Vacancy Service | Вакансии и поиск предложений. Запрос и ответ передаются как сообщения Protobuf. | gRPC / HTTP2 |
| API Gateway | Hiring Service | Отклики, интервью и офферы. Запрос и ответ передаются как сообщения Protobuf. | gRPC / HTTP2 |
| API Gateway | Assessment Service | Задания и результаты оценки. Запрос и ответ передаются как сообщения Protobuf. | gRPC / HTTP2 |
| API Gateway | Communication Service | Диалоги и сообщения. Запрос и ответ передаются как сообщения Protobuf. | gRPC / HTTP2 |
| API Gateway | Identity Service | Проверка пользователя и сессии. Запрос и ответ передаются как сообщения Protobuf. | gRPC / HTTP2 |
| API Gateway | Integration Service | Webhooks и настройки интеграций. Запрос и ответ передаются как сообщения Protobuf. | HTTP / gRPC |
| API Gateway | File Service | Загрузка и получение файлов. Запрос и ответ передаются как сообщения Protobuf. | gRPC / HTTP2 |
| API Gateway | Notification Service | Inbox и настройки уведомлений. Запрос и ответ передаются как сообщения Protobuf. | gRPC / HTTP2 |
| Candidate Service | File Service | Загружает и связывает резюме. Запрос и ответ передаются как сообщения Protobuf. | gRPC / HTTP2 |
| Kafka | Hiring Service | Доставляет события найма. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro, порт 9092 |
| Kafka | Notification Service | Доставляет события уведомлений. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro, порт 9092 |
| Kafka | Integration Service | Доставляет события для webhooks. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro, порт 9092 |
| Kafka | Realtime Publisher | Доставляет события для realtime. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro, порт 9092 |
| Kafka | AuthZ Synchronization Service | Доставляет изменения ролей. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro, порт 9092 |
| AuthZ Synchronization Service | OpenFGA | Обновляет отношения доступа. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Candidate Service | OpenFGA | Проверяет доступ к профилям. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Company Service | OpenFGA | Проверяет корпоративные роли. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Vacancy Service | OpenFGA | Проверяет доступ к вакансиям. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Hiring Service | OpenFGA | Проверяет доступ к найму. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Assessment Service | OpenFGA | Проверяет доступ к оцениванию. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Communication Service | OpenFGA | Проверяет доступ к диалогам. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| File Service | OpenFGA | Проверяет доступ к файлам. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Integration Service | OpenFGA | Проверяет управление интеграциями. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Realtime Publisher | Centrifugo | Публикует сообщения каналов. | HTTP API |
| Ingress Controller | Centrifugo | Проксирует realtime-соединения. Данные передаются по постоянному соединению с браузером. | WebSocket |
| Ingress Controller | LiveKit | Проксирует сигнализацию интервью. Данные передаются по постоянному соединению с браузером. | HTTPS / WSS |
| Браузер пользователя | LiveKit | Передаёт аудио и видео интервью. По соединению передаются медиапотоки участников. | WebRTC / ICE / TURN, UDP/TCP |
| Hiring Service | LiveKit | Создаёт комнаты интервью. | HTTPS / API |
| Ingress Controller | API Gateway | Маршрутизирует HR-webhooks. | HTTP / JSON |
| Candidate Service | PostgreSQL | Читает и сохраняет профили. Получатель выполняет чтение или запись в базе данных сервиса. | SQL / TCP, порт 5432 |
| Company Service | PostgreSQL | Читает и сохраняет компании и роли. Получатель выполняет чтение или запись в базе данных сервиса. | SQL / TCP, порт 5432 |
| Vacancy Service | PostgreSQL | Читает и сохраняет вакансии. Получатель выполняет чтение или запись в базе данных сервиса. | SQL / TCP, порт 5432 |
| Hiring Service | PostgreSQL | Читает и сохраняет отклики, интервью и офферы. Получатель выполняет чтение или запись в базе данных сервиса. | SQL / TCP, порт 5432 |
| Assessment Service | PostgreSQL | Читает и сохраняет задания и оценки. Получатель выполняет чтение или запись в базе данных сервиса. | SQL / TCP, порт 5432 |
| Communication Service | PostgreSQL | Читает и сохраняет диалоги и сообщения. Получатель выполняет чтение или запись в базе данных сервиса. | SQL / TCP, порт 5432 |
| Identity Service | PostgreSQL | Читает и сохраняет пользователей и сессии. Получатель выполняет чтение или запись в базе данных сервиса. | SQL / TCP, порт 5432 |
| Integration Service | PostgreSQL | Читает и сохраняет inbox, outbox и доставки. Получатель выполняет чтение или запись в базе данных сервиса. | SQL / TCP, порт 5432 |
| Notification Service | PostgreSQL | Читает и сохраняет шаблоны и Inbox. Получатель выполняет чтение или запись в базе данных сервиса. | SQL / TCP, порт 5432 |
| OpenFGA | PostgreSQL | Хранит модели и отношения доступа. Получатель выполняет чтение или запись в базе данных сервиса. | SQL / TCP, порт 5432 |

<!-- /excerpt -->

Оформление повторяет deployment участника Member 2: направление слева направо, синие блоки приложений, отдельные узлы исполнения, общий узел СУБД и подписанные связи. Вложенность показывает размещение экземпляров приложений на worker nodes production-кластера. `Container` — экземпляр именованного логического контейнера L2 (например, Hiring Service), а не Kubernetes Pod или группа сервисов. Для прикладных сервисов кратность Pod указана в технической подписи блока; для Web Application, Ingress, LiveKit и stateful workloads Pod дополнительно раскрыт через `Deployment_Node`. Kubernetes namespace обозначен границей `Boundary`. Кратность `× N` показывает число экземпляров на весь пул, а не на каждый узел.

На application-виде каждый сервис показан отдельно. **Web Application разворачивается внутри Kubernetes**, в `careeria-apps`, как Next.js-приложение в `Web Application Pods × 2..10`. Оно обслуживает страницы и клиентский JavaScript; BFF выполняется в серверной части того же Next.js-приложения. На устройстве пользователя показан браузер (`Node`), который загружает интерфейс из кластера. Ingress Controller показан как исполняемый proxy-контейнер внутри Pod, публичный Load Balancer — как инфраструктурный `Node`.

API Gateway связан со всеми десятью прикладными API: Candidate, Company, Vacancy, Hiring, Assessment, Communication, Identity, Integration, File и Notification Service. Candidate Service обращается к File Service для работы с резюме. Восемь сервисов, выполняющих проверки доступа к ресурсам, связаны с OpenFGA; AuthZ Synchronization Service обновляет отношения доступа по событиям Kafka. Kafka также связан с остальными потребителями событий, а Realtime Publisher — с Centrifugo. Сигнализация LiveKit отделена от прямого медиатрафика браузера. Kubernetes Deployment, Service и HPA остаются управляющими объектами и аннотациями. На основной схеме в контуре данных развёрнут один общий PostgreSQL-кластер: к нему показаны SQL-подключения девяти прикладных сервисов и OpenFGA. File Service использует MinIO; его связь с объектным хранилищем раскрыта на следующем виде. Внешние системы показаны на L0/L1.

### Платформенные сервисы и постоянные данные

<!-- excerpt -->

Сервисы используют общий кластер PostgreSQL, а File Service хранит файлы в MinIO. Debezium читает изменения из журнала PostgreSQL и публикует события в Kafka.

Представление раскрывает окружение хранения и поддержки платформы. Сервисы подключаются к общему кластеру PostgreSQL со своими ролями, File Service работает с MinIO через S3 API. Debezium, Kafka и Registry обеспечивают событийный обмен. Для OpenFGA, Centrifugo и LiveKit показаны зависимости от баз данных и отдельных контуров Redis.

```plantuml
@startuml
!include architecture/c4/plantuml/deployment/kubernetes-data.puml
@enduml
```

<p align="center">Рисунок - Платформенные сервисы и постоянные данные</p>

<p align="right">Таблица - Элементы. Платформенные сервисы и постоянные данные</p>

| Элемент | Тип или технология | Назначение и ответственность |
|---|---|---|
| Инфраструктура Careeria | Production Kubernetes Cluster | Размещает вложенные экземпляры компонентов в окружении Production Kubernetes Cluster. |
| Пул узлов приложений × 3..N | Linux / container runtime | Размещает вложенные экземпляры компонентов в окружении Linux / container runtime. |
| OpenFGA Pods × 2 | Kubernetes Pod | Размещает вложенные экземпляры компонентов в окружении Kubernetes Pod. |
| OpenFGA | OpenFGA, PostgreSQL | Сервис проверки доступа на основе отношений между субъектом и ресурсом. Доменные сервисы запрашивают решение о доступе, а обработчик синхронизации обновляет отношения по событиям платформы. |
| Schema Registry Pods × 2 | Kubernetes Pod | Размещает вложенные экземпляры компонентов в окружении Kubernetes Pod. |
| Schema Registry | Apicurio Registry / Avro | Хранилище версий схем событий Avro на базе Apicurio Registry. Издатели и потребители получают схемы для сериализации и чтения доменных сообщений. |
| Kafka Connect Pods × 2 | Kubernetes Pod / distributed workers | Размещает вложенные экземпляры компонентов в окружении Kubernetes Pod / distributed workers. |
| Debezium | Debezium PostgreSQL Connector | Компонент захвата зафиксированных изменений PostgreSQL. Читает transactional outbox через журнал WAL и публикует доменные события в Kafka. |
| Identity Service | Go / gRPC · Pods × 2..10 | Identity Service регистрирует пользователей, проверяет учётные данные, выпускает токены и управляет сессиями. Для входа через внешнего провайдера сервис поддерживает OIDC. |
| Notification Service | Go / Kafka consumer · Pods × 2..10 | Notification Service принимает команды и события из Kafka, выбирает шаблон и создаёт задачу доставки. Обработчики сохраняют уведомления в личном кабинете или передают их провайдерам email, push и SMS. |
| File Service | Go / gRPC / MinIO SDK · Pods × 2..10 | File Service выдаёт временную ссылку на загрузку в объектное хранилище, проверяет загруженную версию и связывает её с ресурсом платформы. Привязка определяет, какую версию файла можно скачать. |
| Centrifugo | Centrifugo / WebSocket · Pods × 2..N | Сервис подключений браузеров и каналов обновления интерфейса. Принимает публикации Realtime Publisher и доставляет сообщения авторизованным подписчикам по WebSocket. |
| Hiring Service | Go / gRPC · Pods × 2..10 | Hiring Service ведёт отклик от первого этапа отбора до предложения о работе. Компоненты откликов, интервью и предложений согласуют переходы между этапами и сохраняют историю в Hiring DB. |
| Realtime Publisher | Go / Kafka consumer · Pods × 2..8 | Realtime Publisher читает события из Kafka, определяет канал получателя и публикует сообщение в Centrifugo. Centrifugo доставляет его в браузер по WebSocket. |
| Медиа-узлы × 2..N | Linux / host networking | Размещает вложенные экземпляры компонентов в окружении Linux / host networking. |
| LiveKit Pods × 2..N | Kubernetes Pod / один Pod на узел | Размещает вложенные экземпляры компонентов в окружении Kubernetes Pod / один Pod на узел. |
| LiveKit | LiveKit / WebRTC SFU | Медиасервер для интервью и Live Coding. Сервисы управляют комнатами через API, а браузеры участников передают аудио, видео и демонстрацию экрана через WebRTC. |
| Узлы данных × 3..N | Linux / persistent volumes | Размещает вложенные экземпляры компонентов в окружении Linux / persistent volumes. |
| Общий PostgreSQL-кластер | Kubernetes StatefulSet / primary + 2 replicas / PVC | Размещает вложенные экземпляры компонентов в окружении Kubernetes StatefulSet / primary + 2 replicas / PVC. |
| PostgreSQL | PostgreSQL | Общая реляционная СУБД платформы. Сервисы используют собственные логические базы или схемы и роли подключения, сохраняя данные через SQL. Постоянные тома обеспечивают сохранение данных экземпляров кластера. |
| Kafka Pods × 3 | Kubernetes Pod / StatefulSet / PVC на экземпляр | Размещает вложенные экземпляры компонентов в окружении Kubernetes Pod / StatefulSet / PVC на экземпляр. |
| Kafka | Apache Kafka / KRaft | Событийная шина платформы. Хранит доменные сообщения в топиках и передаёт их группам потребителей для независимой асинхронной обработки. |
| Identity Redis Pods | Kubernetes Pod / HA topology / PVC | Размещает вложенные экземпляры компонентов в окружении Kubernetes Pod / HA topology / PVC. |
| Identity Redis | Redis | Краткоживущие транзакции авторизации, сессионные шаги и одноразовые authorization codes. |
| Centrifugo Redis Pods | Kubernetes Pod / HA topology / PVC | Размещает вложенные экземпляры компонентов в окружении Kubernetes Pod / HA topology / PVC. |
| Centrifugo Redis | Redis | Broker сообщений и presence; отдельный контур от Identity и LiveKit. |
| LiveKit Redis Pods | Kubernetes Pod / HA topology / PVC | Размещает вложенные экземпляры компонентов в окружении Kubernetes Pod / HA topology / PVC. |
| LiveKit Redis | Redis | Общая координация комнат и media-узлов LiveKit. |
| MinIO Pods × N | Kubernetes Pod / erasure coding / PVC на диск | Размещает вложенные экземпляры компонентов в окружении Kubernetes Pod / erasure coding / PVC на диск. |
| Object Storage | MinIO, S3 API | Приватное объектное хранилище файлов на базе MinIO. File Service управляет версиями и ограниченными ссылками, а браузер передаёт содержимое по выданному разрешению. |

<p align="right">Таблица - Связи. Платформенные сервисы и постоянные данные</p>

| Источник | Получатель | Содержание взаимодействия | Протокол или способ вызова |
|---|---|---|---|
| Debezium | PostgreSQL | Читает WAL и transactional outbox. Получатель выполняет чтение или запись в базе данных сервиса. | PostgreSQL logical replication, порт 5432 |
| Debezium | Kafka | Публикует доменные события. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro, порт 9092 |
| Debezium | Schema Registry | Получает схемы сериализации. | HTTPS / Registry API |
| OpenFGA | PostgreSQL | Читает модели и сохраняет отношения. Получатель выполняет чтение или запись в базе данных сервиса. | SQL / TCP, порт 5432 |
| Schema Registry | PostgreSQL | Хранит схемы и версии. Получатель выполняет чтение или запись в базе данных сервиса. | SQL / TCP, порт 5432 |
| Identity Service | PostgreSQL | Хранит пользователей и сессии. Получатель выполняет чтение или запись в базе данных сервиса. | SQL / TCP, порт 5432 |
| Identity Service | Identity Redis | Хранит одноразовые коды и шаги входа. | Redis / TCP, порт 6379 |
| Notification Service | PostgreSQL | Хранит шаблоны, delivery jobs и Inbox. Получатель выполняет чтение или запись в базе данных сервиса. | SQL / TCP, порт 5432 |
| Kafka | Notification Service | Доставляет события уведомлений. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro, порт 9092 |
| Notification Service | Schema Registry | Получает схемы событий. | HTTPS / Registry API |
| File Service | OpenFGA | Проверяет доступ к файлам. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| File Service | Object Storage | Читает и записывает файлы. Выполняется операция с содержимым или версией объекта в хранилище. | HTTPS / S3 API |
| Centrifugo | Centrifugo Redis | Обменивается сообщениями и presence. | Redis / TCP, порт 6379 |
| LiveKit | LiveKit Redis | Координирует media-узлы и комнаты. | Redis / TCP, порт 6379 |
| Hiring Service | PostgreSQL | Хранит отклики, интервью и офферы. Получатель выполняет чтение или запись в базе данных сервиса. | SQL / TCP, порт 5432 |
| Hiring Service | OpenFGA | Проверяет доступ к найму. Вызов использует модель отношений доступа OpenFGA. | OpenFGA API |
| Hiring Service | LiveKit | Создаёт комнаты интервью. | HTTPS / LiveKit API |
| Kafka | Hiring Service | Доставляет события найма. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro, порт 9092 |
| Hiring Service | Schema Registry | Получает схемы событий. | HTTPS / Registry API |
| Kafka | Realtime Publisher | Доставляет события для realtime. Сообщение передаётся асинхронно через топик Kafka. | Kafka / Avro, порт 9092 |
| Realtime Publisher | Schema Registry | Получает схемы событий. | HTTPS / Registry API |
| Realtime Publisher | Centrifugo | Публикует сообщения каналов. | HTTP API |

<!-- /excerpt -->

Это второй deployment-вид того же production-окружения. PostgreSQL показан как **одна развёрнутая СУБД** — общий PostgreSQL-кластер в namespace `careeria-data`. В обеих схемах изображено одно и то же развёртывание. Прикладные и платформенные сервисы подключаются к нему по SQL/TCP, порт 5432. Логическое владение данными и разделение доступа раскрываются в L2.

На этом виде OpenFGA, Schema Registry, Identity, Hiring и Notification Service подключены к общему PostgreSQL. Identity Service также использует Identity Redis, File Service — OpenFGA и MinIO. Hiring Service управляет комнатами LiveKit; Realtime Publisher получает события Kafka и передаёт сообщения Centrifugo. Centrifugo и LiveKit используют каждый свой Redis-контур.

Debezium подключается к общей СУБД и читает WAL/outbox доменных данных, получает Avro-схемы из Registry и публикует события в Kafka. На уровне развёртывания это связь CDC-коннекторов с PostgreSQL; восемь логических outbox-источников описаны на [событийном представлении L2](../event-driven%28L2%29/index.md). PVC указаны в узлах stateful workloads. Три PostgreSQL Pod — primary и две replicas одного общего кластера СУБД, у каждого Pod собственный постоянный том.

Нотация соответствует назначению [C4 Deployment](https://c4model.com/diagrams/deployment): размещению экземпляров software systems/containers на вложенных deployment nodes с дополнительными infrastructure nodes.

### Состав кластера

<p align="right">Таблица - Состав кластера (Развёртывание в Kubernetes)</p>

| Область | Состав и стартовая конфигурация |
|---|---|
| Control plane | API Server, Scheduler, controller-manager и etcd. Для self-managed варианта — отказоустойчивый control plane; при managed Kubernetes его обслуживает провайдер. Control plane и системные addons на этих двух deployment-видах не детализируются. |
| Application pool | Старт с 3 worker nodes. Здесь работают Ingress controller, приложения, обработчики событий и платформенные сервисы. Пул допускает увеличение числа узлов; верхняя граница определяется бюджетом и ёмкостью инфраструктуры. |
| Data pool | Старт с 3 выделенных worker nodes; workloads размещаются операторами и не масштабируются обычным HPA. Размещение MinIO может потребовать дополнительных узлов в зависимости от выбранной топологии дисков. |
| Media pool | Старт с 2 выделенных worker nodes для LiveKit. В выбранном варианте используется host networking и не более одного LiveKit Pod на узел. Выделение пула позволяет не конкурировать с API за сеть и CPU. |
| Системные компоненты | Metrics Server, DNS, CNI, CSI и сбор метрик. Для прикладных метрик HPA предусматриваются Prometheus и совместимый metrics adapter. |

Предлагаемые namespaces: `careeria-edge`, `careeria-apps`, `careeria-data`, `careeria-media`. Они разделяют конфигурацию и политики доступа; сами по себе namespaces не изолируют сетевой трафик.

### Соответствие контейнерам L2

<p align="right">Таблица - Соответствие контейнерам L2 (Развёртывание в Kubernetes)</p>

| Workload | Контейнеры модели | Размещение |
|---|---|---|
| Пользовательский вход | Web Application (включая BFF), API Gateway | Отдельные Deployment и ClusterIP Service для Web Application и API Gateway; BFF развёртывается и масштабируется вместе с Next.js. Внешний трафик проходит через Ingress controller. |
| Прикладные API | Candidate, Company, Vacancy, Hiring, Assessment, Communication, Identity, Integration, Notification и File Service | 10 независимых Deployments и 10 Service. Для существующих L3 детали доступны через [каталог компонентов](../system-context%28L1%29/full-system%28L2%29/index.md#components). |
| Фоновые обработчики | Notification Service, Realtime Publisher, AuthZ Synchronization Service | Отдельные workloads. API и consumer одного сервиса при необходимости запускаются разными workload-профилями, сохраняя владельца данных. Число Kafka consumers ограничивается числом partitions и поведением consumer group. |
| Realtime | Centrifugo | Deployment и Service; состояние каналов/presence вынесено в Redis broker. Долгоживущие соединения требуют отдельного профиля масштабирования и drain. |
| Авторизация и события | OpenFGA, Apicurio Registry, Debezium / Kafka Connect | Отдельные платформенные Deployments и Service. OpenFGA и Registry подключаются к общей СУБД PostgreSQL со своими ролями; Debezium читает WAL/outbox и публикует события в Kafka. |
| Медиа | LiveKit | Отдельный Deployment на media pool. Сигнализация идёт через HTTPS/WSS, медиатрафик — отдельным маршрутом WebRTC/TURN. |

Распределённые сессии BFF и несколько реплик Next.js требуют общей стратегии хранения сессий, согласованных ключей и политики кэширования. Масштабирование не должно зависеть от локальной памяти конкретного Pod. Точный session backend BFF ещё предстоит выбрать; таблица не назначает ему Identity Redis автоматически.

### Сеть и внешний трафик

Основной путь: `Browser → public Load Balancer → Ingress controller → Next.js (BFF) → API Gateway → Service → ready API Pods`. Вызовы внутренних сервисов используют ClusterIP/DNS; базы данных и Kafka не публикуются в интернет.

Отдельные маршруты Ingress обслуживают WebSocket Centrifugo, сигнализацию LiveKit, входящие HR-webhooks и HTTPS-доступ к S3 API для presigned загрузок и скачиваний. Публичным остаётся ограниченный S3 API endpoint, а не административная консоль MinIO.

Обычный HTTP Ingress не переносит WebRTC-медиатрафик. Для LiveKit отдельно задаются публичные адреса, ICE/TURN, UDP/TCP-порты и правила firewall; способ экспонирования зависит от сети провайдера. Выбранный вариант с `hostNetwork` и ограничением одного Pod на узел соответствует [схеме Kubernetes-развёртывания LiveKit](https://docs.livekit.io/transport/self-hosting/kubernetes/).

## Процесс HPA и масштабирования узлов

Увеличение количества Pod и количества узлов — два разных процесса. [HPA](https://kubernetes.io/docs/concepts/workloads/autoscaling/horizontal-pod-autoscale/) меняет число реплик целевого workload. Если Scheduler не может разместить новые Pod, [Node Autoscaler](https://kubernetes.io/docs/concepts/cluster-administration/node-autoscaling/) при наличии поддерживаемого provisioning API добавляет подходящие worker nodes. На статическом bare-metal кластере без такого API новые узлы нужно предоставлять отдельно.

### Стартовые профили

<p align="right">Таблица - Стартовые профили (Развёртывание в Kubernetes)</p>

| Workload | Стартовый диапазон | Сигнал масштабирования |
|---|---|---|
| Web Application (с BFF), API Gateway | 2–10 Pod **на каждый из двух Deployments** | CPU: целевой уровень 70% от `requests.cpu`. RPS и latency используются для наблюдения; включение custom metrics требует отдельного решения. |
| Прикладные API | 2–10 Pod **на каждый из 10 Deployments** | CPU 70% как начальная настройка; профиль корректируется отдельно для каждого сервиса. |
| Kafka consumers | 2–8 Pod на workload как проектный диапазон | Consumer lag через custom/external metrics. Реальный максимум не выше полезного параллелизма partitions; один workload управляется одним механизмом масштабирования. |
| Centrifugo | Минимум 2 Pod; максимум после замеров | Активные соединения, сообщения и CPU через прикладные метрики. Снижение числа реплик требует drain и корректного переподключения клиентов. |
| LiveKit | Минимум 2 Pod/узла; максимум после замеров | Загрузка комнат, CPU и сети. Масштабирование согласуется с media pool и завершением активных комнат; общий CPU-профиль API сюда не копируется. |
| PostgreSQL, Kafka, Redis, MinIO | По топологии соответствующего оператора | Масштабирование и ребалансировка по правилам продукта; обычный HPA не применяется. |

Для HPA по проценту CPU каждый контейнер целевого Pod должен иметь корректный `requests.cpu`; CPU 70% означает долю от request, а не от всего worker node. [Metrics Server](https://kubernetes-sigs.github.io/metrics-server/) предоставляет ресурсные метрики через Metrics API. Kafka lag и другие прикладные сигналы требуют отдельного источника метрик и adapter; Metrics Server их не создаёт.

Стартовая политика для обычных API: увеличение без дополнительного stabilization window, ограничение темпа роста и `scaleDown.stabilizationWindowSeconds: 300`. Точные policy limits уточняются по времени старта приложения и характеристикам нагрузки. HPA сам не учитывает количество свободных соединений PostgreSQL: для каждого сервиса рассчитывается бюджет `maxReplicas × max DB connections per Pod`, включая резерв на rollout и фоновые задачи.

### Постоянные данные и восстановление

<p align="right">Таблица - Постоянные данные и восстановление (Развёртывание в Kubernetes)</p>

| Система | Размещение и хранение |
|---|---|
| PostgreSQL | Один общий кластер СУБД для прикладных и платформенных сервисов. Оператор управляет primary и двумя replicas; каждый Pod имеет свой PVC. Доступ сервисов разделяется ролями внутри общей СУБД. Политика синхронной репликации и RPO/RTO выбирается отдельно. |
| Kafka | Стартовая топология — 3 узла с KRaft quorum, постоянными томами, replication factor 3 и `min.insync.replicas: 2` для критичных топиков. Выделение controller-узлов и размер partitions определяются при выборе оператора и замерах. |
| Redis | Раздельные инсталляции/контуры для Identity, Centrifugo и LiveKit, собственные PVC и подходящая HA-топология. Требования к долговечности Identity и к transient realtime-состоянию различаются. |
| MinIO | Распределённое S3-хранилище с erasure coding и отдельными PVC для дисков. Количество экземпляров, дисков и StorageClass нельзя выводить из числа application replicas: топология подбирается отдельно. |
| Persistent storage | CSI и StorageClass с подходящими режимами доступа, привязкой томов к зонам и возможностью восстановления после отказа узла. PVC сохраняет данные при пересоздании Pod, но не заменяет репликацию и backup. |

Namespaces, PVC и реплики не обеспечивают резервное копирование автоматически. Нужны PostgreSQL backup/PITR, политики защиты объектов MinIO, восстановления Kafka/Redis по их назначению и регулярные проверки restore. Backup-копии хранятся вне отказного домена production-кластера.

### Доступность и эксплуатация

- Реплики приложений распределяются между узлами и, если инфраструктура позволяет, зонами с помощью [topology spread constraints](https://kubernetes.io/docs/concepts/scheduling-eviction/topology-spread-constraints/). Наличие 3 узлов само по себе не доказывает отказоустойчивость.
- Для обычных API задаются requests/limits, startup/readiness/liveness probes, корректная обработка SIGTERM, graceful shutdown и RollingUpdate с ограничениями на unavailable/surge replicas. [Readiness](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) управляет участием Pod в ready endpoints.
- PodDisruptionBudget ограничивает допустимые добровольные disruptions, например при drain. Он не предотвращает отказ узла и не заменяет настройки rollout; это следует из [модели disruptions Kubernetes](https://kubernetes.io/docs/concepts/workloads/pods/disruptions/).
- NetworkPolicy с подходящим CNI ограничивает связи между workloads. Для сервисов выделяются ServiceAccount и минимальные RBAC-права; TLS-сертификаты и секреты имеют отдельный порядок выдачи и ротации.
- Наблюдаемость охватывает HTTP/gRPC latency, ошибки, насыщение ресурсов, Kafka lag, состояние PVC, replication lag PostgreSQL, WebSocket-соединения и ёмкость media nodes.

## Связанные представления и исходники

- [Контейнеры полной системы (L2)](../system-context%28L1%29/full-system%28L2%29/index.md) — логические приложения, данные и взаимодействия.
- [Коммуникации (L2)](../communication%28L2%29/index.md) — realtime и media-потоки.
- [System Landscape (L0)](../landscape%28L0%29/README.md) — внешние системы и границы ответственности.
- [Исходник размещения приложений](kubernetes-production.puml), [исходник размещения данных](kubernetes-data.puml).

Генерация SVG из корня проекта: `make plantuml-architecture`. Реализация manifests/Helm, выбор контроллеров и операторов, расчёт размеров PVC, ресурсов и верхних границ autoscaling — следующие шаги, а не уже развёрнутая конфигурация.
