# Identity Service — C4 Level 2: Deployment

Диаграмма показывает размещение Identity Service, входного пути приложения и его инфраструктурных зависимостей — PostgreSQL, Redis и Kafka — в **Production Kubernetes**. Контуры `careeria-edge`, `careeria-apps`, `careeria-data` и общий PostgreSQL-кластер соответствуют [сводному deployment платформы](../../../deployment/README.md).

Это целевая топология: Kubernetes-манифестов и Helm chart в проекте пока нет. Порты и состав процесса Identity сверены с кодом; кратность Pod и worker pools взяты из архитектурного проекта.

```plantuml
@startuml
!include architecture/c4/plantuml/members/member4/deployment/member4-deployment.puml
@enduml
```

[Исходник PlantUML](member4-deployment.puml)

## Узлы и размещение контейнеров

| Узел / контур | Среда выполнения | Назначение |
|---|---|---|
| Устройство пользователя | Браузер | Выполняет клиентский JavaScript и отображает интерфейс приложения. |
| Публичный балансировщик | L4 / TCP | Передаёт HTTPS-соединения на Ingress, порт `443`. |
| `careeria-edge`: Ingress | 2 Pod, Nginx / Envoy | TLS-терминация и маршрутизация страниц в Web Application, запросов `/api/*` — в BFF. |
| `careeria-apps`: Web Application | Deployment / HPA / ClusterIP, 2–10 Pod | Next.js / React, SSR и доставка интерфейса; HTTP `3000`. |
| `careeria-apps`: BFF | Deployment / HPA / ClusterIP, 2–10 Pod | Веб-фасад и браузерные сессии; HTTP `8080`. |
| `careeria-apps`: API Gateway | Deployment / HPA / ClusterIP, 2–10 Pod | HTTP API на `8080`, преобразование запросов в gRPC. |
| `careeria-apps`: Identity Service | Deployment / HPA / ClusterIP, 2–10 Pod | Go в `distroless/static-debian12:nonroot`; gRPC и gRPC Health на `50052`. |
| `careeria-data`: PostgreSQL | StatefulSet / оператор, primary и 2 replicas, PVC | Общая СУБД платформы; Identity использует логическую БД `identity` и собственную роль подключения. |
| `careeria-data`: Redis | StatefulSet / HA-топология / PVC | Временные транзакции авторизации и одноразовые authorization codes; порт `6379`. |
| `careeria-data`: Kafka | StatefulSet / KRaft, 3 Pod, PVC | События `identity.events`; `replication.factor=3`, `min.insync.replicas=2`, порт `9092`. |

`Deployment_Node` обозначает место исполнения, `Boundary` — namespace, `Container` — экземпляры логического контейнера L2. Подпись `× N` показывает общее число Pod workload в пуле. Deployment, Service и HPA указаны как управляющие объекты Kubernetes.

PostgreSQL развёрнут общим кластером primary + две replicas, как на сводной схеме платформы. Логическая Identity DB размещается в этом кластере. Redis — обычное Redis-хранилище: здесь показана его роль в хранении временного состояния авторизации. Долговременные пользовательские сессии хранятся в PostgreSQL.

## Сетевые интерфейсы

| Источник → получатель | Назначение | Протокол / порт |
|---|---|---|
| Browser → Load Balancer → Ingress | Загрузка интерфейса и вызовы API | HTTPS / TCP `443`. |
| Ingress → Web Application | Страницы и ресурсы | HTTP `3000`. |
| Ingress → BFF | Запросы `/api/*` | HTTP `8080`. |
| Web Application → BFF | Данные для серверного рендеринга | HTTP / JSON `8080`. |
| BFF → API Gateway | Вызовы HTTP API | HTTP / JSON `8080`. |
| API Gateway → Identity | Registration, Authentication, OAuth и Metadata RPC | gRPC / HTTP2 `50052`. |
| Identity → PostgreSQL | Учётные записи, credentials, профили, челленджи, grants, сессии, token families и outbox | SQL / TCP `5432`. |
| Identity → Redis | Чтение и обновление временного состояния | Redis RESP / TCP `6379`. |
| Identity → Kafka | Публикация событий `identity.events` | Kafka / JSON `9092`. |

Порты `50052` и `8080` взяты из конфигураций Identity и API Gateway. Для Kafka указан внутрикластерный порт `9092` из общей инфраструктуры; `19092` в локальном YAML — опубликованный host-порт.

## Масштабирование и состояние

Стартовый диапазон Identity HPA — 2–10 Pod. Все реплики работают с PostgreSQL и Redis; временный контекст регистрации доступен независимо от обслуживающего Pod. В каждом процессе Identity запускаются gRPC server и outbox worker, который публикует события в Kafka.

При `postgres.max_conns=10` и 10 репликах бюджет соединений Identity составляет до 100 соединений плюс резерв на rollout и миграции. Конкретные параметры HPA и ёмкость общего PostgreSQL уточняются нагрузочными испытаниями.

gRPC Health использует тот же порт `50052`. При завершении Pod приложение получает SIGTERM и останавливает runtime-компоненты с `shutdown.timeout=15s` из текущего YAML. Реплики PostgreSQL, Redis и Kafka используют постоянные тома; восстановление и резервное копирование определяются при реализации инфраструктуры.

Основные источники от корня репозитория: `services/identity/Dockerfile`, `services/identity/configs/config.yaml`, `services/identity/internal/app/{app,messaging}.go`, `services/api-gateway/configs/config.yaml`.

[Ландшафт L0](../landscape/member4-l0-landscape.md) · [Динамика регистрации](../../../dynamic/member4-dynamic.md) · [Контейнеры Member 4](../context%28L1%29/containers%28L2%29/index.md) · [Раздел Member 4](../README.md)
