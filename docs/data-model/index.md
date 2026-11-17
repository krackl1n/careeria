# Модели данных

[Правила конкурентных изменений](concurrency.md) · [Проверка создания и готовности](readiness.md) охватывает все схемы; [аудит нормализации](review.md) подробно рассматривает первые три. В этом разделе находятся DBML-схемы и сведения о хранении данных по сервисам. Описание поведения, сценариев и агрегатов находится в разделе [«Сервисы»](../services/index.md). DBML можно открыть в [dbdiagram.io](https://dbdiagram.io/) или другом редакторе DBML.

| Сервис | Схема | Статус | Диаграммы и таблицы |
|---|---|---|---|
| Identity | `services/identity/db/schema.dbml` | [Аудит существующей схемы](identity/index.md); источник истины — миграции | [Открыть](identity/reference/index.md) |
| Candidate | [schema.dbml](candidate/schema.dbml), [ограничения](candidate/constraints.sql) | Проект | [Открыть](candidate/reference/index.md) |
| Company | [schema.dbml](company/schema.dbml), [ограничения](company/constraints.sql) | [Проектирование сервиса](../services/company/index.md) | [Открыть](company/reference/index.md) |
| Vacancy | [schema.dbml](vacancy/schema.dbml), [ограничения](vacancy/constraints.sql) | [Проектирование сервиса](../services/vacancy/index.md) | [Открыть](vacancy/reference/index.md) |
| Hiring | [schema.dbml](hiring/schema.dbml), [ограничения](hiring/constraints.sql) | [Проектирование сервиса](../services/hiring/index.md) | [Открыть](hiring/reference/index.md) |
| Assessment | [schema.dbml](assessment/schema.dbml), [ограничения](assessment/constraints.sql) | [Проектирование сервиса](../services/assessment/index.md) | [Открыть](assessment/reference/index.md) |
| Communication | [schema.dbml](communication/schema.dbml), [ограничения](communication/constraints.sql) | [Проектирование сервиса](../services/communication/index.md) | [Открыть](communication/reference/index.md) |
| Notification | [schema.dbml](notification/schema.dbml), [ограничения](notification/constraints.sql) | [Проектирование сервиса](../services/notification/index.md) | [Открыть](notification/reference/index.md) |
| Integration | [schema.dbml](integration/schema.dbml), [ограничения](integration/constraints.sql) | [Проектирование сервиса](../services/integration/index.md) | [Открыть](integration/reference/index.md) |
| File | [schema.dbml](file/schema.dbml), [ограничения](file/constraints.sql) | [Проектирование сервиса](../services/file/index.md) | [Открыть](file/reference/index.md) |

Проектные схемы не являются миграциями или полным физическим контрактом. Они показывают ключевые сущности, основные поля и локальные связи. Обязательность, значения по умолчанию, полные индексы, ограничения, хранение истории, retention и стратегии удаления необходимо уточнить до миграций.

Каждый сервис владеет своими данными. FK на диаграммах относятся только к таблицам одной схемы; внешние ID — ссылки по контракту, не межсервисные FK. Таблицы outbox и журналов обработанных событий показаны схематично; перед реализацией их нужно согласовать с общими пакетами и способом доставки.

В File Service предложено хранить метаданные объекта, состояния загрузки и проверок, привязки и задания очистки в собственной PostgreSQL-схеме; байты остаются в Object Storage. Текущая C4-схема явно не определяет File DB. Ключи объектов и служебные сведения принадлежат File Service; другие сервисы хранят `file_id` и необходимые им снимки метаданных.

ER-диаграммы показывают таблицы, поля, ключи и связи в нотации «воронья лапка». Для каждого сервиса приведена одна диаграмма; Identity разделена по предметным областям из-за размера схемы. Таблицы полей можно скопировать из отображаемой страницы в Word. DBML и, для Identity, миграции остаются источниками данных; часть бизнес-семантики требует ручного уточнения.

Обновить справочники после изменения DBML можно из корня репозитория:

```sh
npm install --prefix /tmp/careeria-dbml-tools --no-save @dbml/core@12.1.0 elkjs@0.12.1
NODE_PATH=/tmp/careeria-dbml-tools/node_modules node docs/tools/generate-erd.cjs
NODE_PATH=/tmp/careeria-dbml-tools/node_modules node docs/tools/generate-data-dictionaries.cjs
```

Генераторы перезаписывают `reference/index.md` и `reference/erd*.svg`; пояснения, требующие ручного уточнения, следует хранить в описании сервиса или исходной DBML, а не править только в сгенерированной странице.

## Повторные запросы

Отдельной таблицы ключей идемпотентности нет. Для операций создания в соответствующей создаваемой записи предусмотрены `create_request_id` и `create_request_hash`: UUID уникален в пределах таблицы, а хеш позволяет отличить повтор от другого запроса. Оба поля либо заданы вместе, либо отсутствуют. Перед возвратом найденной записи сервис повторно проверяет полномочия и совпадение хеша. Отсутствие полей допустимо для сценариев без такого контракта и само по себе не защищает их от повтора; до реализации API нужно определить, где они обязательны.

Время защиты совпадает со сроком сохранения самой записи. Повтор не обязан возвращать первоначальное тело ответа, если объект уже изменился или удалён: без отдельной истории команд это нельзя гарантировать. Для изменений и удаления используются естественные ключи, проверка текущего состояния, версии и атомарные условные переходы; семантика повторов каждой команды ещё требует контракта. `processed_events` там, где он предусмотрен, — журнал фактически обработанных событий с результатом, а не таблица ключей HTTP-запросов.
