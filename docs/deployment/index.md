# Развёртывание

В Makefile предусмотрены команды для локального и серверного Compose-окружений:

| Задача | Команда |
|---|---|
| Локальный запуск | `make up` или `make up-d` |
| Серверный запуск | `make up-server` или `make up-server-d` |
| Проверка конфигурации | `make compose-config` или `make compose-config-server` |
| Остановка | `make down` или `make down-server` |
| Логи | `make logs` |

Команды используют `docker-compose.local.yaml` и `docker-compose.server.yaml` соответственно. Перед запуском проверьте наличие этих файлов и настройку окружения в своей версии репозитория.

Архитектура контейнеров описана в [полной системе](../architecture/c4-plant-uml/plantuml/container/full-system.md). Текущая модель C4 не содержит отдельной диаграммы размещения контейнеров на узлах production.
