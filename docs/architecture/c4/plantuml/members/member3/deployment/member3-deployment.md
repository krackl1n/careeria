# Диаграмма развёртывания подсистемы оценки, коммуникаций и realtime-взаимодействия (Level 2: Deployment)

Диаграмма развёртывания C4 (**Deployment Diagram, Level 2**) отражает распределение программных контейнеров подсистем оценки, коммуникаций и realtime-взаимодействия (**Assessment & Communication Domain**, зона ответственности **Member 3** платформы **Careeria**) на физическую и виртуальную вычислительную инфраструктуру, топологию сетевых контуров, узлы отказоустойчивости и параметры сетевой безопасности.

Диаграмма разработана в строгом соответствии с архитектурными стандартами платформы: прикладные сервисы функционируют в кластере **Kubernetes**, базы данных **Assessment DB** и **Communication DB** развернуты на **инстансе БД PostgreSQL** в **StatefulSet** (в режиме Single-Instance без избыточной репликации), брокеры **Apache Kafka (KRaft)** обеспечивают надежную доставку доменных событий, медиа-сервер **LiveKit** и push-сервер **Centrifugo** размещены в собственном медиа/realtime-контуре кластера, а раннеры безопасного исполнения кода кандидатов (**Code Execution Runner**) изолированы на специализированных узлах с рантаймом **gVisor**. Количество подов прикладных сервисов регулируется через HPA.

```plantuml
@startuml
!include architecture/c4/plantuml/members/member3/deployment/member3-deployment.puml
@enduml
```

---

## 1. Спецификация вычислительных узлов и сред выполнения (Deployment Nodes)

| Узел инфраструктуры | Тип среды выполнения | Хостовая платформа / Ресурсы | Размещённые контейнеры и поды | Служебная роль и зона ответственности |
|---|---|---|---|---|
| **Рабочее место пользователя** | Физическое устройство | Windows / macOS / Linux / iOS / Android | `Web Client (Browser Runtime)` | Клиентский интерфейс соискателя (редактор кода, чат, аудио/видео) и нанимающей команды; исполнение React/TypeScript, WebRTC и WebSocket. |
| **Периметральный шлюз (DMZ)** | Виртуальный шлюз | Linux VM / K8s Edge Ingress (2 vCPU, 4 GB RAM) | `Ingress Controller (Nginx / Envoy)` | Точка входа в кластер: SSL/TLS-терминация сертификатов, балансировка L7, защита от DDoS, проксирование HTTP/WSS. |
| **Контур приложений: Web App** | Kubernetes Deployment | K8s Worker Nodes (2 vCPU, 2 GB RAM per pod) | `web-app` (Pods × 2..10, Node.js 20 / Next.js 14) | Серверный рендеринг веб-страниц (SSR), начальная генерация HTML, раздача скомпилированных JS/CSS-бандлов и статических ассетов платформы. Масштабируется через HPA по утилизации CPU. |
| **Контур приложений: BFF** | Kubernetes Deployment | K8s Worker Nodes (2 vCPU, 2 GB RAM per pod) | `bff` (Pods × 2..10, Go 1.23) | Сессионный шлюз, CSRF-защита, проверка `HttpOnly`-cookie `__Host-Careeria-Session`, маршрутизация авторизованных gRPC-вызовов. |
| **Контур приложений: API Gateway** | Kubernetes Pod | K8s Worker Node (2 vCPU, 2 GB RAM) | `api-gateway` (Pods × 2..10, Go 1.23) | Маршрутизация внешних REST/JSON запросов в gRPC-вызовы внутренних сервисов, rate limiting, инжекция W3C Trace Context. |
| **Контур приложений: Assessment Service** | Kubernetes Deployment | K8s Worker Nodes (High CPU, 4 vCPU, 4 GB RAM per pod) | `assessment-service` (Pods × 2..10) | Бизнес-логика банка заданий, управление жизненным циклом сессий испытаний, вычисление скоринговых стратегий (Strategy pattern). |
| **Контур приложений: Communication Service** | Kubernetes Deployment | K8s Worker Nodes (4 vCPU, 4 GB RAM per pod) | `communication-service` (Pods × 2..10) | Управление диалогами, сообщениями, реакциями, вложениями и отметками о прочтении (watermarks). Масштабируется через HPA. |
| **Контур приложений: Code Execution Runner** | Kubernetes Deployment | Dedicated Worker Nodes (gVisor runsc, 4 vCPU, 8 GB RAM) | `code-execution-runner` (Pods × 2..10) | Изолированная среда безопасной компиляции и прогона юнит-тестов на решениях кандидатов без доступа к внешней сети. |
| **Контур приложений: Realtime Publisher** | Kubernetes Deployment | K8s Worker Nodes (2 vCPU, 2 GB RAM per pod) | `realtime-publisher` (Pods × 2..8) | Консьюмер событий Kafka, преобразующий доменные события в форматированные сообщения каналов Centrifugo. |
| **Контур приложений: OpenFGA** | Kubernetes Pod | K8s Worker Node (2 vCPU, 2 GB RAM) | `openfga` (OpenFGA Server × 2) | Высокопроизводительный сервер вычисления графов ReBAC-авторизации отношений (права на прохождение сессии, отправку сообщений). |
| **Контур приложений: Debezium** | Kubernetes Pod | K8s Worker Node (2 vCPU, 4 GB RAM) | `debezium-connector` (Pods × 2) | Агент захвата изменений данных (Change Data Capture), считывающий WAL журналы СУБД PostgreSQL. |
| **Медиа-контур: LiveKit** | Kubernetes Pod | Media Worker Nodes (Host Network, 4 vCPU, 8 GB RAM) | `livekit-server` (Pods × 2..N) | Внутренний кластерный SFU медиа-сервер WebRTC для организации сессий Live Coding и онлайн-интервью. |
| **Медиа-контур: Centrifugo** | Kubernetes Deployment | K8s Worker Nodes (4 vCPU, 4 GB RAM per pod) | `centrifugo` (Pods × 2..5) | Высокопроизводительный realtime WebSocket/SSE сервер, поддерживающий десятки тысяч одновременных клиентских соединений. |
| **Контур данных: PostgreSQL (StatefulSet)** | Kubernetes StatefulSet | High-IOPS Worker Nodes (8 vCPU, 32 GB RAM) | `PostgreSQL (PostgreSQL 16)` | Инстанс БД в StatefulSet (Single-Instance): содержит логические базы данных `Assessment DB` и `Communication DB`, а также таблицы transactional outbox. Порт 5432. |
| **Контур данных: Redis Cluster** | Kubernetes StatefulSet | Memory-Optimized Nodes (2 vCPU, 8 GB RAM) | `Redis Engine (Redis 7.2)` | In-memory хранилище для координации Centrifugo нод (Redis Engine PUB/SUB), хранения presence и кэша сессий. |
| **Контур данных: Apache Kafka Cluster** | Kubernetes StatefulSet | Worker Nodes (3 Pods, 4 vCPU, 16 GB RAM) | `Kafka Brokers (KRaft)` | Отказоустойчивый распределенный журнал доменных событий `assessment.events` и `communication.events` в режиме KRaft. |

---

## 2. Спецификация сетевых интерфейсов, портов и протоколов безопасности

| Исходный узел | Целевой узел | Описание сетевого взаимодействия | Протокол взаимодействия | Сетевой порт | Зона безопасности / Сегмент | Требования к шифрованию |
|---|---|---|---|:---:|---|---|
| Браузер пользователя | Ingress Controller | Загрузка страниц, статических скриптов и вызовы REST API | HTTPS / TLS 1.3 | 443 | Глобальная сеть (Internet) | TLS 1.3, шифр-сьюты ECDHE-RSA-AES256-GCM, HSTS |
| Браузер пользователя | LiveKit Media Server | Аудио/видео стриминг и data-каналы Live Coding | WebRTC / UDP | 7881 | Публичная сеть / Host Network | DTLS-SRTP шифрование медиатрафика |
| Браузер пользователя | Centrifugo Server | Постоянное WebSocket-соединение для мгновенных push-уведомлений и чата | WSS / HTTPS | 8000 | careeria-media namespace | TLS 1.3, авторизация по подписанным JWT |
| Ingress Controller | Web App Pod | Проксирование запросов к SSR-страницам и статике | HTTP / 1.1 | 3000 | Внутрикластерный трафик | ClusterIP, приватная сеть |
| Ingress Controller | BFF Pod | Проксирование API-запросов клиентов (`/api/*`) | HTTP / 1.1 | 8080 | Внутрикластерный трафик | ClusterIP, приватная сеть |
| Ingress Controller | LiveKit Media Server | Сигнализация комнат и WebRTC-рукопожатие | WSS / HTTPS | 7880 | careeria-media namespace | Внутренний mTLS |
| Ingress Controller | Centrifugo Server | Проксирование постоянных WebSocket-соединений | WSS / HTTP 1.1 | 8000 | careeria-media namespace | Внутренний трафик кластера |
| BFF Pod | API Gateway Pod | Авторизованные межсервисные RPC-вызовы | gRPC / HTTP2 | 50051 | Приватная сеть K8s | Внутрикластерный mTLS |
| API Gateway Pod | Assessment Service Pod | Вызовы RPC-методов домена оценки (`SubmitTaskSolution` и др.) | gRPC / HTTP2 | 50055 | careeria-apps namespace | Round-Robin балансировка, mTLS |
| API Gateway Pod | Communication Service Pod | Вызовы RPC-методов чата (`SendMessage`, `ListMessages` и др.) | gRPC / HTTP2 | 50056 | careeria-apps namespace | Round-Robin балансировка, mTLS |
| Assessment Service Pod | OpenFGA Pod | Проверка ReBAC-полномочий на прохождение теста (`can_submit`) | gRPC / HTTP2 | 8081 | Внутренняя сеть K8s | Взаимный mTLS, корпоративный Root CA |
| Communication Service Pod | OpenFGA Pod | Проверка ReBAC-доступа к диалогам | gRPC / HTTP2 | 8081 | Внутренняя сеть K8s | Взаимный mTLS, корпоративный Root CA |
| Assessment Service Pod | Code Execution Runner | Направление кода решений на сборку и запуск тестов | gRPC / HTTP2 | 50060 | careeria-apps namespace | Изолированный сетевой сегмент, mTLS |
| Assessment Service Pod | PostgreSQL (StatefulSet) | Транзакции сохранения сессий, сабмитов и Outbox (Assessment DB) | SQL / TCP | 5432 | Приватная подсеть данных | SCRAM-SHA-256, ClusterIP |
| Communication Service Pod | PostgreSQL (StatefulSet) | Транзакции сохранения сообщений, реакций и Outbox (Communication DB) | SQL / TCP | 5432 | Приватная подсеть данных | SCRAM-SHA-256, ClusterIP |
| PostgreSQL (StatefulSet) | Debezium Connector | Логическая репликация строк таблиц `outbox` | Logical Streaming (pgoutput) | 5432 | Приватная подсеть данных | Слоты `careeria_assessment_slot`, `careeria_comm_slot` |
| Debezium Connector | Kafka Brokers | Публикация Avro-событий в топики доменов | Kafka Protocol / TCP | 9092 | careeria-data namespace | SASL_PLAINTEXT / TLS, acks=all |
| Realtime Publisher Pod | Kafka Brokers | Вычитывание доменных событий для отправки в каналы | Kafka Protocol / TCP | 9092 | careeria-data namespace | Consumer Groups, acks=all |
| Realtime Publisher Pod | Centrifugo Server | Публикация сообщений в клиентские WebSocket-каналы | HTTP / REST API | 8000 | careeria-media namespace | Внутренний API токен (Centrifugo API Key) |
| Centrifugo Server | Redis Engine | Межсерверный Pub/Sub для обмена сообщениями между нодами | Redis Protocol / TCP | 6379 | careeria-data namespace | Авторизация AUTH, приватный кластерный IP |
| Assessment Service Pod | LiveKit Media Server | Управление WebRTC-комнатами и генерация JWT участников | HTTP / REST API | 7880 | careeria-media namespace | Внутрикластерный API токен |

---

## 3. Архитектура развёртывания, масштабирования и отказоустойчивости

### 3.1. Развёртывание прикладных сервисов (Stateless Microservices)
* Микросервисы бизнес-логики `Assessment Service` и `Communication Service`, сервисы шлюзов `Web App`, `BFF` и `API Gateway`, а также брокеры трансляции `Realtime Publisher` разворачиваются как stateless Kubernetes Deployments.
* Горизонтальное масштабирование управляется через **Horizontal Pod Autoscaler (HPA)** на основе утилизации CPU и задержек обработки запросов (Pods × 2..10).
* Трафик балансируется средствами стандартных сервисов Kubernetes (`ClusterIP` и встроенная L4/L7 балансировка).

### 3.2. Безопасность и изоляция исполнения кода (Code Execution Runner)
* Контейнеры `Code Execution Runner` разворачиваются на выделенных воркер-нодах кластера с активированным рантаймом песочницы **gVisor (`runsc`)**.
* Сетевой стек контейнеров песочниц полностью изолирован от внутренней кластерной сети и внешней сети Интернет (`NetworkPolicy: egress: []`), исключая SSRF-атаки, сканирование портов или загрузку вредоносных бинарных файлов со стороны непроверенного кода соискателей.
* На каждый запуск накладываются жесткие cgroups v2 лимиты: CPU quota 1.0, RAM 256 МБ, swap 0 МБ, таймаут 5 секунд.

### 3.3. Инстанс БД PostgreSQL в StatefulSet
* **Инстанс БД:** Базы данных подсистемы Member 3 развёрнуты на общем **инстансе БД (PostgreSQL 16)** в рамках Kubernetes StatefulSet. Внутри данного инстанса БД функционируют две логически независимые базы данных: `Assessment DB` (для тестов, задач и скоринга) и `Communication DB` (для диалогов, сообщений и реакций).
* **Преимущества общего инстанса БД:** Использование одного инстанса БД исключает избыточные накладные расходы на поддержание множества отдельных подов баз данных в кластере, оптимизирует потребление оперативной памяти (Shared Buffers) и упрощает регламентное обслуживание (снапшоты Persistent Volumes, бэкапы pg_dump/pg_wal).
* **Single-Instance без избыточной репликации:** Инстанс PostgreSQL работает как одиночный узел, исключая необходимость сложного кворума Patroni/etcd. Отказоустойчивость шины передачи данных обеспечивается распределенным кластером `Apache Kafka (KRaft)`, а доступность приложений — stateless-масштабированием подов через HPA.
* `Redis Engine` в StatefulSet обеспечивает сверхнизкую латентность межсерверной синхронизации для кластера Centrifugo.

### 3.4. Высоконагруженный Realtime и медиа-контур (Centrifugo & LiveKit)
* **Centrifugo** развёрнут в собственном контуре `careeria-media` с горизонтальным масштабированием подов (Pods × 2..5). Состояние подписок и broadcast-сообщения распределяются между всеми нодами Centrifugo через `Redis Engine PUB/SUB`, гарантируя доставку сообщений кандидатам и рекрутерам независимо от того, к какому поду подключен их веб-сокет.
* **LiveKit** функционирует на выделенных медиа-нодах кластера с режимом `hostNetwork: true`, что исключает накладные расходы Kubernetes NAT и оверлейной сети при передаче высокоинтенсивного UDP-медиатрафика WebRTC (видео, аудио, совместное редактирование кода).

---

[Ландшафт L0](../landscape/member2-l0-landscape.md) · [Контекст L1](../context%28L1%29/index.md) · [Контейнеры L2](../context%28L1%29/containers%28L2%29/index.md) · [Динамический процесс](../../../dynamic/member3-dynamic.md) · [Раздел Member 3](../README.md)
