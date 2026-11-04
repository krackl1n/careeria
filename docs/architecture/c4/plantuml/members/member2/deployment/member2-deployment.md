# Диаграмма развёртывания подсистемы Hiring Service (Level 2: Deployment)

Диаграмма развёртывания C4 (**Deployment Diagram, Level 2**) отражает распределение программных контейнеров подсистемы найма (**Hiring Domain** платформы **Careeria**) на физическую и виртуальную вычислительную инфраструктуру, топологию сетевых контуров, узлы отказоустойчивости и параметры сетевой безопасности.

Диаграмма разработана в соответствии с архитектурными решениями проекта: сервисы запускаются в кластере **Kubernetes**, СУБД **PostgreSQL** и брокеры **Apache Kafka (KRaft)** разворачиваются в **StatefulSet**, а медиа-сервер **LiveKit** развёрнут в собственном медиа-контуре кластера. Количество подов приложений регулируется через HPA без дублирования сущностей на диаграмме.

```plantuml
@startuml
!include architecture/c4/plantuml/members/member2/deployment/member2-deployment.puml
@enduml
```

---

## 1. Архитектура инфраструктурных зон и сетевой периметр

Инфраструктура платформы разделена на изолированные зоны безопасности (**Security Zones**) с детерминированными правилами сетевой фильтрации (Kubernetes `NetworkPolicies` и Cloud Security Groups):

```plantuml
@startuml
!include <C4/C4_Deployment>

title Зонирование сетевой безопасности и инфраструктурные контуры

Deployment_Node(client_tier, "Рабочее место пользователя", "Workstation / Browser") {
    Container(browser, "Браузер клиента", "Next.js SPA", "Пользовательский интерфейс")
}

Deployment_Node(dmz_tier, "Периметральный контур (DMZ / Edge Subnet)", "careeria-edge") {
    Container(ingress_ctrl, "Ingress Controller & Reverse Proxy", "Nginx / Envoy", "SSL/TLS 1.3 Termination, HSTS, Rate Limiting (порт 443)")
}

Deployment_Node(app_tier, "Приватный контур приложений (Application Subnet)", "careeria-apps") {
    Container(bff_pod, "BFF Pod × 2..10", "Go 1.23 / net/http, grpc-go", "Сессионный фасад, валидация HttpOnly кук, CSRF-защита (порт 8080)")
    Container(api_gw, "API Gateway Pod × 2..10", "Go / grpc-gateway", "Маршрутизация REST/gRPC (порт 8080, 50051)")
    Container(hiring_svc, "Hiring Service Pod × 2..10", "Go 1.23+ / K8s Deployment", "Бизнес-логика найма и FSM (порт 50053)")
    Container(openfga_svc, "OpenFGA Server Pod × 2", "OpenFGA Server", "Вычисление ReBAC-графов (порт 8081)")
    Container(debezium_svc, "Debezium Connector Pod × 2", "Kafka Connect / CDC", "Захват изменений таблицы outbox")
}

Deployment_Node(media_tier, "Медиа-контур (Media Subnet)", "careeria-media") {
    Container(livekit_svc, "LiveKit Media Server Pod × 2..N", "LiveKit / WebRTC", "Видеоинтервью и аудио-конференции (WSS 7880, WebRTC 7881)")
}

Deployment_Node(data_tier, "Изолированный контур данных (Data Subnet)", "careeria-data") {
    Deployment_Node(pg_set, "PostgreSQL (StatefulSet)", "Kubernetes StatefulSet") {
        ContainerDb(db_hiring, "Hiring DB", "PostgreSQL 16", "Транзакции найма и outbox (порт 5432)")
    }
    Deployment_Node(kafka_set, "Kafka Cluster (StatefulSet)", "Kubernetes StatefulSet") {
        ContainerQueue(kafka_clu, "Apache Kafka Cluster", "3 Brokers (KRaft)", "Шина доменных событий hiring.events (порт 9092)")
    }
}

Deployment_Node(external_infra, "Внешние корпоративные системы", "External Enterprise Systems") {
    Container(hr_corp_server, "Корпоративная HR-система", "1С:ЗУП / Workday", "Кадровый учет и штатное расписание. Порт 443 (HTTPS)")
}

Rel(browser, ingress_ctrl, "Входящие пользовательские запросы", "HTTPS / TLS 1.3 (порт 443)")
Rel(browser, livekit_svc, "Прямой медиатрафик собеседований", "WebRTC / UDP (порт 7881)")
Rel(ingress_ctrl, bff_pod, "Проксирование клиентских веб-запросов", "HTTP/1.1 (порт 8080)")
Rel(ingress_ctrl, livekit_svc, "Сигнализация сессий", "WSS / HTTPS (порт 7880)")
Rel(bff_pod, api_gw, "Авторизованные межсервисные RPC-вызовы", "gRPC / HTTP2 (порт 50051)")
Rel(api_gw, hiring_svc, "gRPC-вызовы к методам найма", "gRPC / HTTP2 (порт 50053)")
Rel(hiring_svc, openfga_svc, "Проверка прав ReBAC", "gRPC / mTLS (порт 8081)")
Rel(hiring_svc, livekit_svc, "Управление переговорными комнатами", "HTTP / REST API (порт 7880)")
Rel(hiring_svc, db_hiring, "Транзакции записи и чтения", "SQL / TCP (порт 5432)")
Rel(db_hiring, debezium_svc, "Логическая репликация outbox", "pgoutput WAL (порт 5432)")
Rel(debezium_svc, kafka_clu, "Публикация доменных событий", "Kafka Protocol (порт 9092)")
Rel(api_gw, hr_corp_server, "Интеграционные вызовы и вебхуки", "HTTPS / TLS 1.3 (порт 443)")

@enduml
```

---

## 2. Спецификация вычислительных узлов и сред выполнения (Deployment Nodes)

| Узел инфраструктуры | Тип среды выполнения | Хостовая платформа / Ресурсы | Размещённые контейнеры и поды | Служебная роль и зона ответственности |
|---|---|---|---|---|
| **Рабочее место пользователя** | Физическое устройство | Windows / macOS / Linux / iOS / Android | `Web Client (Next.js SPA)` | Интерактивный интерфейс браузера соискателя и сотрудников нанимающей команды. |
| **Периметральный шлюз (DMZ)** | Виртуальный шлюз | Linux VM / K8s Edge Ingress (2 vCPU, 4 GB RAM) | `Ingress Controller (Nginx / Envoy)` | Фронтальный обратный прокси, SSL-терминация сертификатов, защита от DDoS, маршрутизация HTTP(S). |
| **Контур приложений: BFF** | Kubernetes Deployment | K8s Worker Nodes (2 vCPU, 2 GB RAM per pod) | `bff` (Pods × 2..10, Go 1.23) | Управление сессиями, CSRF-защита, валидация cookie `__Host-Careeria-Session`, маршрутизация gRPC к API Gateway. |
| **Контур приложений: API Gateway** | Kubernetes Pod | K8s Worker Node (cgroups v2, 2 vCPU, 2 GB RAM) | `api-gateway` (Go 1.23) | Трансляция внешних REST-запросов в gRPC-вызовы микросервисов, инжекция трассировки OpenTelemetry. |
| **Контур приложений: Hiring Service** | Kubernetes Deployment | K8s Worker Nodes (High CPU, 4 vCPU, 4 GB RAM per pod) | `hiring-service` (Pods × 2..10) | Горизонтально масштабируемый инстанс бизнес-логики найма. Обработка RPC, FSM, фоновые воркеры. Количество реплик масштабируется HPA. |
| **Контур приложений: OpenFGA** | Kubernetes Pod | K8s Worker Node (2 vCPU, 2 GB RAM) | `openfga` (OpenFGA Server) | Высокопроизводительный сервер вычисления графов ReBAC-авторизации отношений доступа. |
| **Контур приложений: Debezium** | Kubernetes Pod | K8s Worker Node (2 vCPU, 4 GB RAM) | `debezium-connector` | Потоковый агент Change Data Capture (CDC), непрерывно считывающий журнал упреждающей записи СУБД. |
| **Медиа-контур: LiveKit** | Kubernetes Pod | Media Worker Nodes (Host Network, 4 vCPU, 8 GB RAM) | `livekit-server` (Pods × 2..N) | Внутренний кластерный медиа-сервер WebRTC для организации онлайн-собеседований (без использования внешнего SaaS). |
| **Контур данных: PostgreSQL (StatefulSet)** | Kubernetes StatefulSet | High-IOPS Worker Nodes (8 vCPU, 32 GB RAM) | `Hiring DB (PostgreSQL 16)` | База данных сервиса найма в StatefulSet. Без межсерверной репликации (Single-Instance). |
| **Контур данных: Apache Kafka Cluster** | Kubernetes StatefulSet | Worker Nodes (3 Pods, 4 vCPU, 16 GB RAM) | `Kafka Brokers (KRaft)` | Отказоустойчивый распределённый журнал доменных событий. Хранение топика `hiring.events` в StatefulSet. |
| **Корпоративная HR-система** | Корпоративный контур | On-Premise Enterprise VM / Private Cloud | `1С:ЗУП 8.3 / Workday HCM` | Главная система кадрового учёта компании-работодателя. |

---

## 3. Спецификация сетевых интерфейсов, портов и протоколов безопасности

| Исходный узел | Целевой узел | Описание сетевого взаимодействия | Протокол взаимодействия | Сетевой порт | Зона безопасности / Сегмент | Требования к шифрованию |
|---|---|---|---|:---:|---|---|
| Браузер пользователя | Ingress Controller | Доставка статики фронтенда, REST-запросы к API платформы | HTTPS / TLS 1.3 | 443 | Глобальная сеть (Internet) | TLS 1.3, шифр-сьюты ECDHE-RSA-AES256-GCM, HSTS |
| Браузер пользователя | LiveKit Media Server | Аудио- и видеопотоки собеседования, демонстрация экрана | WebRTC / UDP | 7881 | Публичная сеть / Host Network | DTLS-SRTP шифрование медиатрафика |
| Ingress Controller | BFF Pod | Проксирование клиентских веб-запросов | HTTP / 1.1 | 8080 | Приватная сеть K8s Cluster | Внутрикластерный трафик (ClusterIP) |
| BFF Pod | API Gateway Pod | Авторизованные межсервисные RPC-вызовы | gRPC / HTTP2 | 50051 | Приватная сеть K8s Cluster | Внутрикластерный mTLS / ClusterIP |
| Ingress Controller | LiveKit Media Server | Сигнализация сессий видеоинтервью | WSS / HTTPS | 7880 | careeria-media namespace | Внутренний трафик кластера |
| API Gateway Pod | Hiring Service Pod | Вызовы RPC-методов домена найма (`SubmitFeedback`, `ChangeStage` и др.) | gRPC / HTTP2 | 50053 | K8s Service Mesh / Overlay | Балансировка Round-Robin, mTLS |
| Hiring Service Pod | OpenFGA Pod | Проверка ReBAC-полномочий на действия над объектами найма | gRPC / HTTP2 | 8081 | Внутренняя сеть K8s | Взаимный mTLS, корпоративный Root CA |
| Hiring Service Pod | LiveKit Media Server | Создание переговорных комнат, выпуск JWT-токенов участников | HTTP / REST API | 7880 | careeria-media namespace | Внутрикластерный API токен |
| Hiring Service Pod | PostgreSQL (StatefulSet) | Транзакции бизнес-логики: создание откликов, смена этапов, Outbox | SQL / TCP | 5432 | Приватная подсеть данных (StatefulSet) | Внутрикластерное соединение, SCRAM-SHA-256 |
| PostgreSQL | Debezium Connector | Логическая репликация строк служебной таблицы `outbox` | Logical Streaming (pgoutput) | 5432 | Приватная подсеть данных | Логический слот репликации `careeria_hiring_slot` |
| Debezium Connector | Kafka Brokers | Публикация сериализованных Avro-событий найма в топик `hiring.events` | Kafka Protocol / TCP | 9092 | Сеть брокеров сообщений (StatefulSet) | SASL_PLAINTEXT / TLS, acks=all |
| API Gateway Pod | Корпоративная HR-система | Передача кадровых статусов и данных кандидатов | HTTPS / REST API | 443 | Защищённый корпоративный канал | Корпоративный VPN (IPsec) / mTLS, API Key |

---

## 4. Архитектура развёртывания и отказоустойчивости

### 4.1. Развёртывание приложений (Hiring Service Deployment)
* Контейнер `Hiring Service` развёрнут как стандартный Kubernetes Deployment.
* На диаграмме экземпляр сервиса отображается однократно с указанием кратности (`Pods × 2..10`).
* Масштабирование регулируется через **Horizontal Pod Autoscaler (HPA)** по метрикам CPU и задержки RPC-запросов.
* Балансировка входящего трафика между подами обеспечивается стандартным объектом `Kubernetes Service` (ClusterIP).

### 4.2. Контур данных в StatefulSet (PostgreSQL)
* База данных `Hiring DB` развёрнута в **Kubernetes StatefulSet**.
* **Отказ от репликации СУБД:** В соответствии с принятым решением, база данных работает как одиночный экземпляр (**Single-Instance**), исключая избыточную сложность настройки Patroni HA, физической репликации WAL и DCS-кворума (etcd). Отсутствие необходимости распределенного консенсуса для БД существенно упрощает эксплуатацию и обслуживание, тогда как отказоустойчивость и надежность шины данных обеспечиваются репликацией брокеров `Kafka KRaft` (3 реплики, `replication.factor=3`), а высокая доступность прикладных сервисов — шардированием и динамическим масштабированием подов stateless-приложений через HPA.

### 4.3. Брокер сообщений в StatefulSet (Apache Kafka KRaft)
* Кластер брокеров Kafka развёрнут в **StatefulSet** из 3 подов в режиме **KRaft** (без ZooKeeper).
* Топик `hiring.events` конфигурируется с `replication.factor=3`, гарантируя отказоустойчивость шины событий.

### 4.4. Внутренний медиа-контур (LiveKit)
* Медиа-сервер **LiveKit** разворачивается внутри собственного контура инфраструктуры (namespace `careeria-media`) на выделенных worker-узлах с host networking, а не через внешние SaaS-сервисы.
* Это исключает зависимость от сторонних облачных провайдеров и снижает сетевые задержки при проведении собеседований.

---

[Ландшафт L0](../landscape/member2-l0-landscape.md) · [Контекст L1](../context%28L1%29/index.md) · [Контейнеры L2](../context%28L1%29/containers%28L2%29/index.md) · [Динамический процесс](../../../dynamic/member2-dynamic.md) · [Раздел Member 2](../README.md)
