# Диаграмма развёртывания подсистемы Candidate Service и Vacancy Service (Level 2: Deployment)

Диаграмма развёртывания C4 (**Deployment Diagram, Level 2**) отражает физическое и виртуальное распределение программных контейнеров подсистемы кандидатов и вакансий (**Candidate & Vacancy Domain** платформы **Careeria**) на вычислительную инфраструктуру кластера **Production Kubernetes**, топологию сетевых контуров, параметры сетевой фильтрации и узлы отказоустойчивости.

Диаграмма разработана в соответствии с общекомандными архитектурными решениями проекта: прикладные сервисы запускаются в кластере **Kubernetes** под управлением HPA, реляционная СУБД **PostgreSQL 16** разворачивается в изолированном **StatefulSet как единый инстанс (Single-Instance без межсерверной репликации)** с логическим разделением баз данных `candidate_db` и `vacancy_db`, а объектное хранилище **MinIO** и брокеры **Apache Kafka (KRaft)** функционируют в выделенных **StatefulSet** контура данных. Загрузка файлов резюме реализована по паттерну **Presigned URL** — браузер клиента загружает файл напрямую в Object Storage, минуя API-уровень платформы.

```plantuml
@startuml
!include architecture/c4/plantuml/members/member1/deployment/member1-deployment.puml
@enduml
```

---

## 1. Спецификация вычислительных узлов и сред выполнения (Deployment Nodes)

| Узел инфраструктуры | Тип среды выполнения | Хостовая платформа / Ресурсы | Размещённые контейнеры и поды | Служебная роль и зона ответственности |
|---|---|---|---|---|
| **Рабочее место пользователя** | Физическое устройство | Windows / macOS / Linux / iOS / Android | `Web Client (Browser Runtime)` | Графический интерфейс соискателя и рекрутера, выполнение клиентского кода React, гидратация DOM-дерева, прямая загрузка файлов резюме в Object Storage по presigned URL. |
| **Периметральный шлюз (DMZ)** | Виртуальный шлюз (`careeria-edge`) | Linux VM / K8s Edge Ingress (2 vCPU, 4 GB RAM) | `Ingress Controller (Nginx / Envoy)` | Фронтальный обратный прокси, SSL-терминация сертификатов TLS 1.3, защита от DDoS, rate limiting и маршрутизация HTTP(S)-трафика. |
| **Контур приложений: Web App** | Kubernetes Deployment (`careeria-apps`) | K8s Worker Nodes (2 vCPU, 2 GB RAM per pod) | `web-app` (Pods × 2..10, Node.js 20 / Next.js 14) | Серверный рендеринг веб-страниц профилей и вакансий (SSR), начальная генерация HTML, раздача скомпилированных JS/CSS-бандлов и статических ассетов. Масштабируется через HPA по утилизации CPU. |
| **Контур приложений: BFF** | Kubernetes Deployment (`careeria-apps`) | K8s Worker Nodes (2 vCPU, 2 GB RAM per pod) | `bff` (Pods × 2..10, Go 1.23) | Управление сессиями, CSRF-защита, валидация защищённой cookie `__Host-Careeria-Session`, агрегация и маршрутизация gRPC-вызовов к API Gateway. |
| **Контур приложений: API Gateway** | Kubernetes Deployment (`careeria-apps`) | K8s Worker Nodes (cgroups v2, 2 vCPU, 2 GB RAM) | `api-gateway` (Pods × 2..10, Go 1.23) | Трансляция внешних REST-запросов в gRPC-вызовы микросервисов, верификация сессий, инжекция контекста `ctxmeta` и трассировки OpenTelemetry. |
| **Контур приложений: Candidate Service** | Kubernetes Deployment (`careeria-apps`) | K8s Worker Nodes (2 vCPU, 4 GB RAM per pod) | `candidate-service` (Pods × 2..10, Go 1.23+) | Горизонтально масштабируемый инстанс бизнес-логики управления профилями и резюме (`Candidate Profile Service`, `Resume Service`, ролевое маскирование `MaskingStrategy`, фоновый `Resume Cleanup Worker`). Количество реплик масштабируется HPA. |
| **Контур приложений: Vacancy Service** | Kubernetes Deployment (`careeria-apps`) | K8s Worker Nodes (2 vCPU, 4 GB RAM per pod) | `vacancy-service` (Pods × 2..10, Go 1.23+) | Горизонтально масштабируемый инстанс бизнес-логики жизненного цикла вакансий (FSM `DRAFT` / `PUBLISHED` / `CLOSED` / `ARCHIVED`, проверка `PublishingPolicy`, CQRS-разделение команд и поисковых запросов). Масштабируется через HPA. |
| **Контур приложений: File Service** | Kubernetes Deployment (`careeria-apps`) | K8s Worker Nodes (1 vCPU, 2 GB RAM per pod) | `file-service` (Pods × 2..10, Go 1.23) | Управление файловыми метаданными, генерация presigned URL для загрузки и скачивания резюме, валидация MIME-типов и размера, запуск antivirus hook. |
| **Контур приложений: OpenFGA** | Kubernetes Deployment (`careeria-apps`) | K8s Worker Nodes (2 vCPU, 2 GB RAM) | `openfga` (Pods × 2, OpenFGA Server) | Высокопроизводительный сервер вычисления графов ReBAC-авторизации отношений доступа к профилям, резюме и вакансиям. |
| **Контур приложений: Debezium** | Kubernetes Deployment (`careeria-apps`) | K8s Worker Nodes (2 vCPU, 4 GB RAM) | `debezium-connector` (Pods × 2) | Потоковый агент Change Data Capture (CDC), непрерывно считывающий журнал упреждающей записи СУБД (WAL) из таблиц `outbox` логических баз `candidate_db` и `vacancy_db`. |
| **Контур данных: PostgreSQL (StatefulSet)** | Kubernetes StatefulSet (`careeria-data`) | High-IOPS Worker Nodes (8 vCPU, 32 GB RAM, NVMe PVC) | `PostgreSQL 16 (candidate_db, vacancy_db)` | Единый инстанс СУБД в StatefulSet без межсерверной репликации (Single-Instance). Содержит изолированные логические БД `candidate_db` и `vacancy_db` с таблицами бизнес-агрегатов, `resume_cleanup_jobs` и `transactional outbox`. |
| **Контур данных: MinIO Object Storage** | Kubernetes StatefulSet (`careeria-data`) | High-Bandwidth Worker Nodes (4 vCPU, 8 GB RAM) | `MinIO (S3-compatible)` | Приватное объектное хранилище бинарных файлов резюме, аватаров и вложений. Поддерживает S3 API, presigned URL и bucket lifecycle policies. |
| **Контур данных: Apache Kafka Cluster** | Kubernetes StatefulSet (`careeria-data`) | Worker Nodes (3 Pods, 4 vCPU, 16 GB RAM) | `Kafka Brokers (3 Nodes, KRaft)` | Отказоустойчивый распределённый журнал доменных событий. Хранение топиков `candidate.events` и `vacancy.events` в StatefulSet. |

---

## 2. Спецификация сетевых интерфейсов, портов и протоколов безопасности

| Исходный узел | Целевой узел | Описание сетевого взаимодействия | Протокол взаимодействия | Сетевой порт | Зона безопасности / Сегмент | Требования к шифрованию |
|---|---|---|---|:---:|---|---|
| Браузер пользователя | Ingress Controller | Доставка веб-страниц, статики и вызовы API платформы | HTTPS / TLS 1.3 | 443 | Глобальная сеть (Internet) | TLS 1.3, шифр-сьюты ECDHE-RSA-AES256-GCM, HSTS |
| Браузер пользователя | MinIO Object Storage | Прямая загрузка файла резюме по presigned PUT URL (TTL 15 мин) | HTTPS / S3 API | 9000 | Публичная сеть / DMZ | TLS 1.3, подпись AWS Signature V4, URL с ограниченным сроком действия |
| Ingress Controller | Web App Pod | Проксирование запросов за страницами профилей и вакансий (SSR) | HTTP / 1.1 | 3000 | Приватная сеть K8s Cluster | Внутрикластерный трафик (ClusterIP) |
| Ingress Controller | BFF Pod | Проксирование клиентских запросов к API платформы (`/api/*`) | HTTP / 1.1 | 8080 | Приватная сеть K8s Cluster | Внутрикластерный трафик (ClusterIP) |
| BFF Pod | API Gateway Pod | Авторизованные межсервисные RPC-вызовы с проверенной сессией | gRPC / HTTP2 | 50051 | Приватная сеть K8s Cluster | Внутрикластерный mTLS / ClusterIP |
| API Gateway Pod | Candidate Service Pod | Вызовы RPC-методов домена профилей (`GetProfile`, `UpdateSkills`, `RequestResumeUpload`, `ConfirmResumeUpload`) | gRPC / HTTP2 | 50052 | K8s Service Mesh / Overlay | Балансировка Round-Robin, mTLS |
| API Gateway Pod | Vacancy Service Pod | Вызовы RPC-методов домена вакансий (`CreateDraftVacancy`, `PublishVacancy`, `ListVacancies`) | gRPC / HTTP2 | 50054 | K8s Service Mesh / Overlay | Балансировка Round-Robin, mTLS |
| Candidate Service Pod | OpenFGA Pod | Проверка ReBAC-полномочий на действия над профилями и резюме (`can_view`, `can_upload_resume`) | gRPC / HTTP2 | 8081 | Внутренняя сеть K8s | Взаимный mTLS, корпоративный Root CA |
| Vacancy Service Pod | OpenFGA Pod | Проверка ReBAC-полномочий на действия над вакансиями (`can_create_vacancy`, `can_publish`) | gRPC / HTTP2 | 8081 | Внутренняя сеть K8s | Взаимный mTLS, корпоративный Root CA |
| Candidate Service Pod | PostgreSQL (StatefulSet) | Транзакции в логическую БД `candidate_db`: профили, версии резюме, задачи очистки и `outbox` | SQL / TCP | 5432 | Приватная подсеть данных (StatefulSet) | Внутрикластерное соединение, SCRAM-SHA-256 |
| Vacancy Service Pod | PostgreSQL (StatefulSet) | Транзакции в логическую БД `vacancy_db`: вакансии, требования, навыки и `outbox` | SQL / TCP | 5432 | Приватная подсеть данных (StatefulSet) | Внутрикластерное соединение, SCRAM-SHA-256 |
| Candidate Service Pod | File Service Pod | gRPC-вызовы для генерации presigned URL, сверки статуса загрузки и удаления файлов резюме | gRPC / HTTP2 | 50055 | Приватная сеть K8s Cluster | Внутрикластерный mTLS |
| File Service Pod | MinIO Object Storage | Генерация presigned URL, проверка метаданных объектов и управление файлами через S3 API | S3 API / HTTPS | 9000 | Приватная подсеть данных | Внутрикластерное S3-соединение, Access Key / Secret Key |
| PostgreSQL (StatefulSet) | Debezium Connector | Логическая репликация строк служебных таблиц `outbox` из `candidate_db` и `vacancy_db` | Logical Streaming (pgoutput) | 5432 | Приватная подсеть данных | Логические слоты репликации `careeria_candidate_slot`, `careeria_vacancy_slot` |
| Debezium Connector | Kafka Brokers | Публикация сериализованных Avro-событий в топики `candidate.events` и `vacancy.events` | Kafka Protocol / TCP | 9092 | Сеть брокеров сообщений (StatefulSet) | SASL_PLAINTEXT / TLS, `acks=all` |

---

## 3. Топология отказоустойчивости и сетевой периметр

### 3.1. Горизонтальное автомасштабирование (HPA)
* Все микросервисы бизнес-логики (`Candidate Service`, `Vacancy Service`), файлового сервиса (`File Service`), уровня представления (`Web App Pod`) и сессионного слоя (`BFF Pod`) развёрнуты как stateless-приложения под управлением **Horizontal Pod Autoscaler (HPA)** с кратностью от `2` до `10` реплик по метрикам утилизации CPU и задержки ответов.
* Балансировка входящей нагрузки между подами обеспечивается нативными виртуальными IP-адресами `Kubernetes Service` (ClusterIP).
* Безопасная параллельная работа фоновых воркеров `Resume Cleanup Worker` в нескольких репликах `Candidate Service` гарантируется механизмом неблокирующего захвата строк PostgreSQL `SELECT ... FOR UPDATE SKIP LOCKED` в таблице `resume_cleanup_jobs`.

### 3.2. Отказ от избыточной репликации СУБД (Single-Instance PostgreSQL в едином StatefulSet)
* СУБД **PostgreSQL 16** развёрнута в изолированном неймспейсе `careeria-data` в рамках **единого Kubernetes StatefulSet как одиночный инстанс (Single-Instance)**.
* Изоляция хранилищ по принципу **Database-per-Service** реализована на уровне отдельных логических баз данных (`candidate_db` и `vacancy_db`) внутри единого инстанса PostgreSQL: `Candidate Service` и `Vacancy Service` используют независимые пулы соединений `pgx` и собственные учётные записи СУБД, не имея перекрёстного доступа к таблицам друг друга.
* Отказ от создания отдельного StatefulSet под каждую БД, а также отказ от Master-Replica репликации через `Patroni` и распределённого DCS-кворума (`etcd`) обоснован снижением эксплуатационной сложности и накладных расходов по памяти: сохранность данных обеспечивается персистентными томами PVC, а гарантия доставки доменных событий перенесена на паттерн **Transactional Outbox** и отказоустойчивый кластер **Apache Kafka KRaft**.

### 3.3. Отказоустойчивая шина событий (Kafka KRaft в StatefulSet)
* Кластер брокеров Kafka развёрнут в **StatefulSet** из 3 узлов в режиме **KRaft** (без ZooKeeper).
* Топики `candidate.events` и `vacancy.events` сконфигурированы с фактором репликации `replication.factor=3` и подтверждением записи `acks=all`, гарантируя нулевую потерю доменных событий при сбое любого отдельного узла кластера.

### 3.4. Объектное хранилище (MinIO / S3 API) и прямая загрузка по Presigned URL
* **MinIO** развёрнут в `careeria-data` как **StatefulSet** с persistent volumes для надёжного хранения файлов резюме (PDF/DOCX) и аватаров (порт 9000 — S3 API, порт 9001 — Console) и поддержкой `bucket lifecycle policies`.
* Загрузка резюме реализована по паттерну **Presigned URL**: браузер запрашивает временную подписанную ссылку через `BFF` → `API Gateway` → `Candidate Service` → `File Service` → MinIO генерирует `presigned PUT URL` с TTL 15 минут и подписью AWS Signature V4.
* Браузер загружает бинарный файл **напрямую в MinIO**, минуя API-уровень платформы — это исключает двойную пересылку трафика и буферизацию тяжёлых файлов в оперативной памяти Go-сервисов. После завершения выгрузки браузер вызывает `ConfirmResumeUpload`, а `Candidate Service` атомарно сохраняет версию резюме в статусе `READY` и записывает событие в таблицу `outbox`.

---

[Ландшафт L0](../landscape/member1-l0-landscape.md) · [Контекст L1](../context%28L1%29/index.md) · [Контейнеры L2](../context%28L1%29/containers%28L2%29/index.md) · [Динамический процесс загрузки резюме](../../../dynamic/member1-dynamic.md) · [Раздел Member 1](../README.md)