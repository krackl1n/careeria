# Развёртывание

Целевой production-вариант — [Kubernetes-кластер с HPA и постоянными томами](../architecture/c4/plantuml/deployment/README.md). На странице показаны размещение компонентов, масштабирование Pod и worker nodes, хранение данных и эксплуатационные требования. Это проектная схема; Kubernetes-манифесты пока не добавлены.

В Makefile предусмотрены команды для локального и серверного Compose-окружений:

| Задача | Команда |
|---|---|
| Локальный запуск | `make up` или `make up-d` |
| Серверный запуск | `make up-server` или `make up-server-d` |
| Проверка конфигурации | `make compose-config` или `make compose-config-server` |
| Остановка | `make down` или `make down-server` |
| Логи | `make logs` |

Команды используют `docker-compose.local.yaml` и `docker-compose.server.yaml` соответственно. Перед запуском проверьте наличие этих файлов и настройку окружения в своей версии репозитория.

Архитектура контейнеров описана в [полной системе](../architecture/c4/plantuml/system-context%28L1%29/full-system%28L2%29/index.md). Размещение контейнеров в production описано в [диаграммах Kubernetes](../architecture/c4/plantuml/deployment/README.md).
