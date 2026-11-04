# Развёртывание в Kubernetes

Целевая схема production-развёртывания Careeria. Приложения и хранилища работают **внутри Kubernetes**; PostgreSQL, Kafka, Redis и MinIO используют постоянные тома. Это архитектурный проект: Kubernetes-манифестов и Helm chart в репозитории пока нет.

Число узлов, реплик и пороги ниже — стартовые параметры для проектирования. Они не являются результатом расчёта производительности и уточняются нагрузочными испытаниями. Облачный провайдер, версия Kubernetes, реализация Ingress controller, CSI и конкретные операторы пока не выбраны.

## Размещение компонентов

```plantuml
@startuml
!include architecture/c4/plantuml/deployment/kubernetes-production.puml
@enduml
```

Оформление повторяет deployment участника Member 2: направление слева направо, синие блоки приложений, отдельные узлы исполнения, общий узел СУБД и подписанные связи. Вложенность показывает размещение экземпляров приложений на worker nodes production-кластера. `Container` — экземпляр именованного логического контейнера L2 (например, Hiring Service), а не Kubernetes Pod или группа сервисов. Для прикладных сервисов кратность Pod указана в технической подписи блока; для Web Application, Ingress, LiveKit и stateful workloads Pod дополнительно раскрыт через `Deployment_Node`. Kubernetes namespace обозначен границей `Boundary`. Кратность `× N` показывает число экземпляров на весь пул, а не на каждый узел.

На application-виде каждый сервис показан отдельно. **Web Application разворачивается внутри Kubernetes**, в `careeria-apps`, как Next.js-приложение в `Web Application Pods × 2..10`. Оно обслуживает страницы, доставляет клиентский JavaScript и обращается к BFF. На устройстве пользователя показан браузер (`Node`), который загружает интерфейс из кластера. Ingress Controller показан как исполняемый proxy-контейнер внутри Pod, публичный Load Balancer — как инфраструктурный `Node`.

API Gateway связан со всеми десятью прикладными API: Candidate, Company, Vacancy, Hiring, Assessment, Communication, Identity, Integration, File и Notification Service. Candidate Service обращается к File Service для работы с резюме. Восемь сервисов, выполняющих проверки доступа к ресурсам, связаны с OpenFGA; AuthZ Synchronization Service обновляет отношения доступа по событиям Kafka. Kafka также связан с остальными потребителями событий, а Realtime Publisher — с Centrifugo. Сигнализация LiveKit отделена от прямого медиатрафика браузера. Kubernetes Deployment, Service и HPA остаются управляющими объектами и аннотациями. На основной схеме в контуре данных развёрнут один общий PostgreSQL-кластер: к нему показаны SQL-подключения девяти прикладных сервисов и OpenFGA. File Service использует MinIO; его связь с объектным хранилищем раскрыта на следующем виде. Внешние системы показаны на L0/L1.

### Платформенные сервисы и постоянные данные

```plantuml
@startuml
!include architecture/c4/plantuml/deployment/kubernetes-data.puml
@enduml
```

Это второй deployment-вид того же production-окружения. PostgreSQL показан как **одна развёрнутая СУБД** — общий PostgreSQL-кластер в namespace `careeria-data`. В обеих схемах изображено одно и то же развёртывание. Прикладные и платформенные сервисы подключаются к нему по SQL/TCP, порт 5432. Логическое владение данными и разделение доступа раскрываются в L2.

На этом виде OpenFGA, Schema Registry, Identity, Hiring и Notification Service подключены к общему PostgreSQL. Identity Service также использует Identity Redis, File Service — OpenFGA и MinIO. Hiring Service управляет комнатами LiveKit; Realtime Publisher получает события Kafka и передаёт сообщения Centrifugo. Centrifugo и LiveKit используют каждый свой Redis-контур.

Debezium подключается к общей СУБД и читает WAL/outbox доменных данных, получает Avro-схемы из Registry и публикует события в Kafka. На уровне развёртывания это связь CDC-коннекторов с PostgreSQL; восемь логических outbox-источников описаны на [событийном представлении L2](../event-driven%28L2%29/index.md). PVC указаны в узлах stateful workloads. Три PostgreSQL Pod — primary и две replicas одного общего кластера СУБД, у каждого Pod собственный постоянный том.

Нотация соответствует назначению [C4 Deployment](https://c4model.com/diagrams/deployment): размещению экземпляров software systems/containers на вложенных deployment nodes с дополнительными infrastructure nodes.

### Состав кластера

| Область | Состав и стартовая конфигурация |
|---|---|
| Control plane | API Server, Scheduler, controller-manager и etcd. Для self-managed варианта — отказоустойчивый control plane; при managed Kubernetes его обслуживает провайдер. Control plane и системные addons на этих двух deployment-видах не детализируются; их роли в масштабировании показаны отдельно на activity diagram. |
| Application pool | Старт с 3 worker nodes. Здесь работают Ingress controller, приложения, обработчики событий и платформенные сервисы. Пул допускает увеличение числа узлов; верхняя граница определяется бюджетом и ёмкостью инфраструктуры. |
| Data pool | Старт с 3 выделенных worker nodes; workloads размещаются операторами и не масштабируются обычным HPA. Размещение MinIO может потребовать дополнительных узлов в зависимости от выбранной топологии дисков. |
| Media pool | Старт с 2 выделенных worker nodes для LiveKit. В выбранном варианте используется host networking и не более одного LiveKit Pod на узел. Выделение пула позволяет не конкурировать с API за сеть и CPU. |
| Системные компоненты | Metrics Server, DNS, CNI, CSI и сбор метрик. Для прикладных метрик HPA предусматриваются Prometheus и совместимый metrics adapter. |

Предлагаемые namespaces: `careeria-edge`, `careeria-apps`, `careeria-data`, `careeria-media`. Они разделяют конфигурацию и политики доступа; сами по себе namespaces не изолируют сетевой трафик.

### Соответствие контейнерам L2

| Workload | Контейнеры модели | Размещение |
|---|---|---|
| Пользовательский вход | Web Application, BFF, API Gateway | По отдельному Deployment и ClusterIP Service. Внешний трафик проходит через Ingress controller. |
| Прикладные API | Candidate, Company, Vacancy, Hiring, Assessment, Communication, Identity, Integration, Notification и File Service | 10 независимых Deployments и 10 Service. Для существующих L3 детали доступны через [каталог компонентов](../system-context%28L1%29/full-system%28L2%29/index.md#components); File Service пока раскрыт на L2. |
| Фоновые обработчики | Notification Service, Realtime Publisher, AuthZ Synchronization Service | Отдельные workloads. API и consumer одного сервиса при необходимости запускаются разными workload-профилями, сохраняя владельца данных. Число Kafka consumers ограничивается числом partitions и поведением consumer group. |
| Realtime | Centrifugo | Deployment и Service; состояние каналов/presence вынесено в Redis broker. Долгоживущие соединения требуют отдельного профиля масштабирования и drain. |
| Авторизация и события | OpenFGA, Apicurio Registry, Debezium / Kafka Connect | Отдельные платформенные Deployments и Service. OpenFGA и Registry подключаются к общей СУБД PostgreSQL со своими ролями; Debezium читает WAL/outbox и публикует события в Kafka. |
| Медиа | LiveKit | Отдельный Deployment на media pool. Сигнализация идёт через HTTPS/WSS, медиатрафик — отдельным маршрутом WebRTC/TURN. |

Распределённые сессии BFF и несколько реплик Next.js требуют общей стратегии хранения сессий, согласованных ключей и политики кэширования. Масштабирование не должно зависеть от локальной памяти конкретного Pod. Точный session backend BFF ещё предстоит выбрать; таблица не назначает ему Identity Redis автоматически.

### Сеть и внешний трафик

Основной путь: `Browser → public Load Balancer → Ingress controller → Web/BFF → API Gateway → Service → ready API Pods`. Вызовы внутренних сервисов используют ClusterIP/DNS; базы данных и Kafka не публикуются в интернет.

Отдельные маршруты Ingress обслуживают WebSocket Centrifugo, сигнализацию LiveKit, входящие HR-webhooks и HTTPS-доступ к S3 API для presigned загрузок и скачиваний. Публичным остаётся ограниченный S3 API endpoint, а не административная консоль MinIO.

Обычный HTTP Ingress не переносит WebRTC-медиатрафик. Для LiveKit отдельно задаются публичные адреса, ICE/TURN, UDP/TCP-порты и правила firewall; способ экспонирования зависит от сети провайдера. Выбранный вариант с `hostNetwork` и ограничением одного Pod на узел соответствует [схеме Kubernetes-развёртывания LiveKit](https://docs.livekit.io/transport/self-hosting/kubernetes/).

## Процесс HPA и масштабирования узлов

```plantuml
@startuml
!include architecture/c4/plantuml/deployment/kubernetes-autoscaling.puml
@enduml
```

Этот вид — UML Activity diagram процесса управления, а не C4 Deployment: метрики, изменение replicas, решение Scheduler и provisioning не являются местами исполнения приложений.

Увеличение количества Pod и количества узлов — два разных процесса. [HPA](https://kubernetes.io/docs/concepts/workloads/autoscaling/horizontal-pod-autoscale/) меняет число реплик целевого workload. Если Scheduler не может разместить новые Pod, [Node Autoscaler](https://kubernetes.io/docs/concepts/cluster-administration/node-autoscaling/) при наличии поддерживаемого provisioning API добавляет подходящие worker nodes. На статическом bare-metal кластере без такого API новые узлы нужно предоставлять отдельно.

### Стартовые профили

| Workload | Стартовый диапазон | Сигнал масштабирования |
|---|---|---|
| Web, BFF, API Gateway | 2–10 Pod **на каждый Deployment** | CPU: целевой уровень 70% от `requests.cpu`. RPS и latency используются для наблюдения; включение custom metrics требует отдельного решения. |
| Прикладные API | 2–10 Pod **на каждый из 10 Deployments** | CPU 70% как начальная настройка; профиль корректируется отдельно для каждого сервиса. |
| Kafka consumers | 2–8 Pod на workload как проектный диапазон | Consumer lag через custom/external metrics. Реальный максимум не выше полезного параллелизма partitions; один workload управляется одним механизмом масштабирования. |
| Centrifugo | Минимум 2 Pod; максимум после замеров | Активные соединения, сообщения и CPU через прикладные метрики. Снижение числа реплик требует drain и корректного переподключения клиентов. |
| LiveKit | Минимум 2 Pod/узла; максимум после замеров | Загрузка комнат, CPU и сети. Масштабирование согласуется с media pool и завершением активных комнат; общий CPU-профиль API сюда не копируется. |
| PostgreSQL, Kafka, Redis, MinIO | По топологии соответствующего оператора | Масштабирование и ребалансировка по правилам продукта; обычный HPA не применяется. |

Для HPA по проценту CPU каждый контейнер целевого Pod должен иметь корректный `requests.cpu`; CPU 70% означает долю от request, а не от всего worker node. [Metrics Server](https://kubernetes-sigs.github.io/metrics-server/) предоставляет ресурсные метрики через Metrics API. Kafka lag и другие прикладные сигналы требуют отдельного источника метрик и adapter; Metrics Server их не создаёт.

Стартовая политика для обычных API: увеличение без дополнительного stabilization window, ограничение темпа роста и `scaleDown.stabilizationWindowSeconds: 300`. Точные policy limits уточняются по времени старта приложения и характеристикам нагрузки. HPA сам не учитывает количество свободных соединений PostgreSQL: для каждого сервиса рассчитывается бюджет `maxReplicas × max DB connections per Pod`, включая резерв на rollout и фоновые задачи.

### Постоянные данные и восстановление

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
- [Исходник размещения приложений](kubernetes-production.puml), [исходник размещения данных](kubernetes-data.puml) и [исходник процесса autoscaling](kubernetes-autoscaling.puml).

Генерация SVG из корня проекта: `make plantuml-architecture`. Реализация manifests/Helm, выбор контроллеров и операторов, расчёт размеров PVC, ресурсов и верхних границ autoscaling — следующие шаги, а не уже развёрнутая конфигурация.
