# Диаграмма развёртывания подсистемы Candidate Service и Vacancy Service (Level 2: Deployment)

Диаграмма развёртывания C4 (**Deployment Diagram, Level 2**) отражает распределение программных контейнеров подсистемы кандидатов и вакансий (**Candidate & Vacancy Domain** платформы **Careeria**) на физическую и виртуальную вычислительную инфраструктуру, топологию сетевых контуров, узлы отказоустойчивости и параметры сетевой безопасности.

Диаграмма разработана в соответствии с архитектурными решениями проекта: сервисы запускаются в кластере **Kubernetes**, СУБД **PostgreSQL** и объектное хранилище **MinIO**, а также брокеры **Apache Kafka (KRaft)** разворачиваются в **StatefulSet**. Загрузка файлов резюме реализована по паттерну **Presigned URL** — браузер клиента загружает файл напрямую в Object Storage, минуя API-уровень платформы.

```plantuml
@startuml
!include architecture/c4/plantuml/members/member1/deployment/member1-deployment.puml
@enduml
```

---

## 1. Спецификация вычислительных узлов и сред выполнения (Deployment Nodes)

| Узел инфраструктуры | Тип среды выполнения | Хостовая платформа / Ресурсы | Размещённые контейнеры и поды | Служебная роль и зона ответственности |
|---|---|---|---|---|
| **Рабочее место пользователя** | Физическое устройство | Windows / macOS / Linux / iOS / Android | `Web Client (Browser Runtime)` | Графический интерфейс кандидата и рекрутера, выполнение клиентского кода React, загрузка файлов резюме напрямую в Object Storage по presigned URL. |
| **Периметральный шлюз (DMZ)** | Виртуальный шлюз | Linux VM / K8s Edge Ingress (2 vCPU, 4 GB RAM) | `Ingress Controller (Nginx / Envoy)` | Фронтальный обратный прокси, SSL-терминация TLS 1.3, защита от DDoS, маршрутизация HTTP(S)-трафика. |
| **Контур приложений: Web App** | Kubernetes Deployment | K8s Worker Nodes (2 vCPU, 2 GB RAM per pod) | `web-app` (Pods × 2..10, Node.js 20 / Next.js 14) | Серверный рендеринг страниц профилей и вакансий (SSR), начальная генерация HTML, раздача JS/CSS-бандлов. Масштабируется через HPA по утилизации CPU. |
| **Контур приложений: BFF** | Kubernetes Deployment | K8s Worker Nodes (2 vCPU, 2 GB RAM per pod) | `bff` (Pods × 2..10, Go 1.23) | Управление сессиями, CSRF-защита, валидация HttpOnly-cookie, агрегация и маршрутизация gRPC-запросов к API Gateway. |
| **Контур приложений: API Gateway** | Kubernetes Pod | K8s Worker Node (cgroups v2, 2 vCPU, 2 GB RAM) | `api-gateway` (Go 1.23) | Трансляция внешних REST-запросов в gRPC-вызовы микросервисов, rate limiting, инжекция трассировки OpenTelemetry. |
| **Контур приложений: Candidate Service** | Kubernetes Deployment | K8s Worker Nodes (2 vCPU, 4 GB RAM per pod) | `candidate-service` (Pods × 2..10) | Горизонтально масштабируемый инстанс бизнес-логики управления профилями. Обработка RPC, координация загрузки резюме через File Service. |
| **Контур приложений: Vacancy Service** | Kubernetes Deployment | K8s Worker Nodes (2 vCPU, 4 GB RAM per pod) | `vacancy-service` (Pods × 2..10) | Горизонтально масштабируемый инстанс бизнес-логики жизненного цикла вакансий. Создание, публикация и архивирование вакансий. |
| **Контур приложений: File Service** | Kubernetes Deployment | K8s Worker Nodes (1 vCPU, 2 GB RAM per pod) | `file-service` (Pods × 2..10, Go 1.23) | Управление файловыми метаданными, генерация presigned URL для загрузки и скачивания, MIME-валидация и antivirus hook. |
| **Контур приложений: OpenFGA** | Kubernetes Pod | K8s Worker Node (2 vCPU, 2 GB RAM) | `openfga` (OpenFGA Server) | Высокопроизводительный сервер вычисления графов ReBAC-авторизации отношений доступа к профилям и вакансиям. |
| **Контур приложений: Debezium** | Kubernetes Pod | K8s Worker Node (2 vCPU, 4 GB RAM) | `debezium-connector` | Потоковый агент Change Data Capture (CDC), считывающий журнал WAL из outbox-таблиц Candidate DB и Vacancy DB. |
| **Контур данных: Candidate DB (StatefulSet)** | Kubernetes StatefulSet | High-IOPS Worker Nodes (4 vCPU, 16 GB RAM) | `Candidate DB (PostgreSQL 16)` | База данных сервиса профилей в StatefulSet. Хранит профили, статусы поиска, метаданные резюме и outbox-записи. |
| **Контур данных: Vacancy DB (StatefulSet)** | Kubernetes StatefulSet | High-IOPS Worker Nodes (4 vCPU, 16 GB RAM) | `Vacancy DB (PostgreSQL 16)` | База данных сервиса вакансий в StatefulSet. Хранит вакансии, требования, навыки и outbox-записи. |
| **Контур данных: MinIO Object Storage** | Kubernetes StatefulSet | High-Bandwidth Worker Nodes (4 vCPU, 8 GB RAM) | `MinIO (S3-compatible)` | Объектное хранилище файлов резюме, аватаров и вложений. Поддерживает S3 API, presigned URL и bucket lifecycle policies. |
| **Контур данных: Apache Kafka Cluster** | Kubernetes StatefulSet | Worker Nodes (3 Pods, 4 vCPU, 16 GB RAM) | `Kafka Brokers (KRaft)` | Отказоустойчивый распределённый журнал доменных событий. Хранение топиков `candidate.events` и `vacancy.events` в StatefulSet. |

---

## 2. Спецификация сетевых интерфейсов, портов и протоколов безопасности

| Исходный узел | Целевой узел | Описание сетевого взаимодействия | Протокол взаимодействия | Сетевой порт | Зона безопасности / Сегмент | Требования к шифрованию |
|---|---|---|---|:---:|---|---|
| Браузер пользователя | Ingress Controller | Доставка веб-страниц, статики и вызовы API платформы | HTTPS / TLS 1.3 | 443 | Глобальная сеть (Internet) | TLS 1.3, шифр-сьюты ECDHE-RSA-AES256-GCM, HSTS |
| Браузер пользователя | MinIO Object Storage | Прямая загрузка файла резюме по presigned PUT URL (TTL 15 мин) | HTTPS / S3 API | 9000 | Публичная сеть / DMZ | TLS 1.3, подпись AWS Signature V4, URL с ограниченным сроком действия |
| Ingress Controller | Web App Pod | Проксирование запросов за страницами профилей и вакансий (SSR) | HTTP / 1.1 | 3000 | Приватная сеть K8s Cluster | Внутрикластерный трафик (ClusterIP) |
| Ingress Controller | BFF Pod | Проксирование клиентских API-запросов (/api/*) | HTTP / 1.1 | 8080 | Приватная сеть K8s Cluster | Внутрикластерный трафик (ClusterIP) |
| BFF Pod | API Gateway Pod | Авторизованные межсервисные RPC-вызовы с проверенной сессией | gRPC / HTTP2 | 50051 | Приватная сеть K8s Cluster | Внутрикластерный mTLS / ClusterIP |
| API Gateway Pod | Candidate Service Pod | Вызовы RPC-методов домена профилей (`GetProfile`, `UpdateProfile`, `RequestResumeUpload` и др.) | gRPC / HTTP2 | 50052 | K8s Service Mesh / Overlay | Балансировка Round-Robin, mTLS |
| API Gateway Pod | Vacancy Service Pod | Вызовы RPC-методов домена вакансий (`CreateVacancy`, `PublishVacancy`, `SearchVacancies` и др.) | gRPC / HTTP2 | 50054 | K8s Service Mesh / Overlay | Балансировка Round-Robin, mTLS |
| API Gateway Pod | File Service Pod | Запрос presigned URL для загрузки и получение метаданных файлов | gRPC / HTTP2 | 50055 | K8s Service Mesh / Overlay | Внутрикластерный mTLS |
| Candidate Service Pod | OpenFGA Pod | Проверка ReBAC-полномочий на доступ к профилям (`can_view_profile`, `can_upload_resume`) | gRPC / HTTP2 | 8081 | Внутренняя сеть K8s | Взаимный mTLS, корпоративный Root CA |
| Vacancy Service Pod | OpenFGA Pod | Проверка ReBAC-полномочий на доступ к вакансиям (`can_create_vacancy`, `can_publish_vacancy`) | gRPC / HTTP2 | 8081 | Внутренняя сеть K8s | Взаимный mTLS, корпоративный Root CA |
| Candidate Service Pod | Candidate DB | Транзакции профилей, резюме и записи в таблицу outbox | SQL / TCP | 5432 | Приватная подсеть данных (StatefulSet) | Внутрикластерное соединение, SCRAM-SHA-256 |
| Vacancy Service Pod | Vacancy DB | Транзакции вакансий, требований и записи в таблицу outbox | SQL / TCP | 5432 | Приватная подсеть данных (StatefulSet) | Внутрикластерное соединение, SCRAM-SHA-256 |
| Candidate Service Pod | File Service Pod | gRPC-вызов для генерации presigned URL и подтверждения загрузки файла | gRPC / HTTP2 | 50055 | Приватная сеть K8s Cluster | Внутрикластерный mTLS |
| File Service Pod | MinIO Object Storage | Генерация presigned URL, запись/чтение файлов через S3 API | S3 API / HTTPS | 9000 | Приватная подсеть данных | Внутрикластерное S3-соединение, Access Key / Secret Key |
| Candidate DB | Debezium Connector | Логическая репликация строк из outbox-таблицы `candidate.events` | Logical Streaming (pgoutput) | 5432 | Приватная подсеть данных | Логический слот репликации `careeria_candidate_slot` |
| Vacancy DB | Debezium Connector | Логическая репликация строк из outbox-таблицы `vacancy.events` | Logical Streaming (pgoutput) | 5432 | Приватная подсеть данных | Логический слот репликации `careeria_vacancy_slot` |
| Debezium Connector | Kafka Brokers | Публикация сериализованных Avro-событий в топики `candidate.events` и `vacancy.events` | Kafka Protocol / TCP | 9092 | Сеть брокеров сообщений (StatefulSet) | SASL_PLAINTEXT / TLS, acks=all |

---

## 3. Архитектура развёртывания и отказоустойчивости

### 3.1. Развёртывание приложений (Application Deployments)
* Контейнеры бизнес-логики `Candidate Service` и `Vacancy Service`, а также сервисы уровня представления (`Web App` и `BFF`) разворачиваются как stateless Kubernetes Deployments с горизонтальным автомасштабированием через **Horizontal Pod Autoscaler (HPA)** по метрикам утилизации CPU и задержки ответов (Pods × 2..10).
* `File Service` также разворачивается как stateless Deployment с HPA — он не хранит состояния и только координирует операции с MinIO.
* На диаграмме каждый инстанс сервиса отображается однократно с указанием диапазона реплик (`Pods × 2..10`).
* Балансировка входящего трафика между подами обеспечивается стандартными объектами `Kubernetes Service` (ClusterIP).

### 3.2. Контур данных: Candidate DB и Vacancy DB (PostgreSQL)
* Базы данных `Candidate DB` и `Vacancy DB` развёрнуты в отдельных **Kubernetes StatefulSet**.
* Каждый сервис (Candidate Service и Vacancy Service) работает со своей изолированной базой данных по принципу **Database per Service** — это обеспечивает независимость схем и возможность независимого масштабирования хранилищ.
* **Transactional Outbox Pattern**: каждая бизнес-транзакция атомарно записывает данные и outbox-запись в одну транзакцию СУБД — это гарантирует консистентность событий без распределённых транзакций.

### 3.3. Брокер сообщений в StatefulSet (Apache Kafka KRaft)
* Кластер брокеров Kafka развёрнут в **StatefulSet** из 3 подов в режиме **KRaft** (без ZooKeeper).
* Топики `candidate.events` и `vacancy.events` конфигурируются с `replication.factor=3`, гарантируя отказоустойчивость шины событий при выходе одного брокера из строя.

### 3.4. Объектное хранилище (MinIO / S3 API)
* **MinIO** развёрнут как StatefulSet с persistent volumes для надёжного хранения файлов резюме и аватаров.
* Поддерживаются `bucket lifecycle policies` для автоматического удаления временных файлов (незавершённые загрузки, устаревшие версии резюме).
* MinIO доступен внутри кластера на порту 9000 (S3 API) и 9001 (Console).

### 3.5. Presigned URL и прямая загрузка файлов
* Загрузка резюме реализована по паттерну **Presigned URL**: браузер запрашивает временную подписанную ссылку через API → `Candidate Service` → `File Service` → MinIO генерирует presigned PUT URL с TTL 15 минут и подписью AWS Signature V4.
* Браузер загружает файл **напрямую в MinIO**, минуя API-уровень платформы — это снижает нагрузку на сервисы и устраняет необходимость буферизации бинарных данных в памяти бизнес-логики.
* После успешной загрузки браузер подтверждает факт загрузки через `ConfirmResumeUpload` RPC — Candidate Service записывает метаданные файла и публикует событие через Outbox.

---

[Ландшафт L0](../landscape/member1-l0-landscape.md) · [Контекст L1](../context%28L1%29/index.md) · [Контейнеры L2](../context%28L1%29/containers%28L2%29/index.md) · [Динамический процесс загрузки резюме](../../../dynamic/member1-dynamic.md) · [Раздел Member 1](../README.md)

