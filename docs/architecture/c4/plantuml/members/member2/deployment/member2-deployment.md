# Диаграмма развёртывания подсистемы Hiring Service (Level 2: Deployment)

Диаграмма развёртывания C4 (**Deployment Diagram, Level 2**) отражает распределение программных контейнеров подсистемы найма (**Hiring Domain** платформы **Careeria**) на физическую и виртуальную вычислительную инфраструктуру, топологию сетевых контуров, узлы отказоустойчивости и параметры сетевой безопасности.

Диаграмма разработана в соответствии с архитектурными решениями проекта: сервисы запускаются в кластере **Kubernetes**, СУБД **PostgreSQL** и брокеры **Apache Kafka (KRaft)** разворачиваются в **StatefulSet**, а медиа-сервер **LiveKit** развёрнут в собственном медиа-контуре кластера. Количество подов приложений регулируется через HPA без дублирования сущностей на диаграмме.

```plantuml
@startuml
!include architecture/c4/plantuml/members/member2/deployment/member2-deployment.puml
@enduml
```

---

## 1. Спецификация вычислительных узлов и сред выполнения (Deployment Nodes)

| Узел инфраструктуры | Тип среды выполнения | Хостовая платформа / Ресурсы | Размещённые контейнеры и поды | Служебная роль и зона ответственности |
|---|---|---|---|---|
| **Рабочее место пользователя** | Физическое устройство | Windows / macOS / Linux / iOS / Android | `Web Client (Browser Runtime)` | Графический интерфейс соискателя и сотрудников нанимающей команды, выполнение клиентского кода React, гидратация DOM-дерева. |
| **Периметральный шлюз (DMZ)** | Виртуальный шлюз | Linux VM / K8s Edge Ingress (2 vCPU, 4 GB RAM) | `Ingress Controller (Nginx / Envoy)` | Фронтальный обратный прокси, SSL-терминация сертификатов, защита от DDoS, маршрутизация HTTP(S). |
| **Контур приложений: Web App** | Kubernetes Deployment | K8s Worker Nodes (2 vCPU, 2 GB RAM per pod) | `web-app` (Pods × 2..10, Node.js 20 / Next.js 14) | Серверный рендеринг веб-страниц (SSR), начальная генерация HTML, раздача скомпилированных JS/CSS-бандлов и статических ассетов платформы. Масштабируется через HPA по утилизации CPU. |
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

## 2. Спецификация сетевых интерфейсов, портов и протоколов безопасности

| Исходный узел | Целевой узел | Описание сетевого взаимодействия | Протокол взаимодействия | Сетевой порт | Зона безопасности / Сегмент | Требования к шифрованию |
|---|---|---|---|:---:|---|---|
| Браузер пользователя | Ingress Controller | Доставка веб-страниц, статики и вызовы API платформы | HTTPS / TLS 1.3 | 443 | Глобальная сеть (Internet) | TLS 1.3, шифр-сьюты ECDHE-RSA-AES256-GCM, HSTS |
| Браузер пользователя | LiveKit Media Server | Аудио- и видеопотоки собеседования, демонстрация экрана | WebRTC / UDP | 7881 | Публичная сеть / Host Network | DTLS-SRTP шифрование медиатрафика |
| Ingress Controller | Web App Pod | Проксирование запросов за страницами и статической сборкой (SSR) | HTTP / 1.1 | 3000 | Приватная сеть K8s Cluster | Внутрикластерный трафик (ClusterIP) |
| Ingress Controller | BFF Pod | Проксирование клиентских запросов к API платформы (/api/*) | HTTP / 1.1 | 8080 | Приватная сеть K8s Cluster | Внутрикластерный трафик (ClusterIP) |
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

## 3. Архитектура развёртывания и отказоустойчивости

### 3.1. Развёртывание приложений (Application Deployments)
* Контейнеры бизнес-логики `Hiring Service`, а также сервисы уровня представления (`Web App` и `BFF`) разворачиваются как stateless Kubernetes Deployments с горизонтальным автомасштабированием через **Horizontal Pod Autoscaler (HPA)** по метрикам утилизации CPU и задержки ответов (Pods × 2..10).
* На диаграмме каждый экземпляр сервиса отображается однократно с указанием диапазона реплик (`Pods × 2..10`).
* Балансировка входящего трафика между подами обеспечивается стандартными объектами `Kubernetes Service` (ClusterIP).

### 3.2. Контур данных в StatefulSet (PostgreSQL)
* База данных `Hiring DB` развёрнута в **Kubernetes StatefulSet**.
* **Отказ от репликации СУБД:** В соответствии с принятым решением, база данных работает как одиночный экземпляр (**Single-Instance**), исключая избыточную сложность настройки Patroni HA, физической репликации WAL и DCS-кворума (etcd). Отсутствие необходимости распределенного консенсуса для БД существенно упрощает эксплуатацию и обслуживание, тогда как отказоустойчивость и надежность шины данных обеспечиваются репликацией брокеров `Kafka KRaft` (3 реплики, `replication.factor=3`), а высокая доступность прикладных сервисов — шардированием и динамическим масштабированием подов stateless-приложений через HPA.

### 3.3. Брокер сообщений в StatefulSet (Apache Kafka KRaft)
* Кластер брокеров Kafka развёрнут в **StatefulSet** из 3 подов в режиме **KRaft** (без ZooKeeper).
* Топик `hiring.events` конфигурируется с `replication.factor=3`, гарантируя отказоустойчивость шины событий.

### 3.4. Внутренний медиа-контур (LiveKit)
* Медиа-сервер **LiveKit** разворачивается внутри собственного контура инфраструктуры (namespace `careeria-media`) на выделенных worker-узлах с host networking, а не через внешние SaaS-сервисы.
* Это исключает зависимость от сторонних облачных провайдеров и снижает сетевые задержки при проведении собеседований.

---

[Ландшафт L0](../landscape/member2-l0-landscape.md) · [Контекст L1](../context%28L1%29/index.md) · [Контейнеры L2](../context%28L1%29/containers%28L2%29/index.md) · [Динамический процесс](../../../dynamic/member2-dynamic.md) · [Раздел Member 2](../README.md)
