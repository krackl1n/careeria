# База данных сервиса Notification

База данных Notification содержит шаблоны уведомлений и их редакции, задания на доставку, историю попыток и сообщения во внутреннем разделе уведомлений. Отдельные записи позволяют учитывать обработанные события и состояние доставки по каждому каналу.

## ER-диаграмма



![ER-диаграмма Notification](erd.svg)

<p align="center">Рисунок - Модель данных сервиса Notification</p>

[Открыть SVG полностью](erd.svg)

## Сущности

<p align="right">Таблица - Сущности сервиса Notification</p>

| Таблица | Псевдоним | Описание | Особенности |
|---|---|---|---|
| notification_templates | Шаблон уведомления | Идентичность и состояние шаблона уведомления. | PK: id |
| template_revisions | Редакция уведомления | Редакция шаблона для конкретного канала и локали. | PK: id |
| delivery_jobs | Доставка уведомления | Задание отправки одному получателю по одному каналу. | PK: id |
| delivery_attempts | Попытка доставки | Результат отдельной попытки отправки уведомления. | PK: id |
| in_app_notifications | Внутреннее уведомление | Сохранённый снимок уведомления для получателя. | PK: id |
| processed_events | Обработанное событие | Запись о результате обработки входящего события потребителем. | PK: consumer, source_event_id |

## Связи

<p align="right">Таблица - Связи сущностей сервиса Notification</p>

| Родительская сущность и ключ | Дочерняя сущность и FK | Дочерних записей на одну родительскую | Родительских записей на одну дочернюю | Комментарий |
|---|---|---|---|---|
| notification_templates (id) | template_revisions (template_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: не задано. |
| template_revisions (id) | delivery_jobs (template_revision_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: не задано. |
| delivery_jobs (id) | delivery_attempts (job_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: не задано. |
| delivery_jobs (id, recipient_id) | in_app_notifications (delivery_job_id, recipient_id) | 0..1 | 1 | Не более одной дочерней записи; ссылка обязательна. Удаление родителя: не задано. |

## Атрибуты

<p align="right">Таблица - Атрибуты сущностей сервиса Notification</p>

<div class="md-typeset__scrollwrap attribute-dictionary"><div class="md-typeset__table"><table><thead><tr><th>Сущность</th><th>Поле</th><th>Тип</th><th>Описание</th></tr></thead><tbody>
<tr><td class="attribute-name" rowspan="6">notification_templates</td><td class="attribute-name">id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Идентификатор записи.</td></tr>
<tr><td class="attribute-name">template_key</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; UNIQUE; Стабильный ключ шаблона.</td></tr>
<tr><td class="attribute-name">status</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Состояние записи; допустимые переходы описаны в проекте сервиса.</td></tr>
<tr><td class="attribute-name">version</td><td class="attribute-type">bigint</td><td class="attribute-description">Обязательно; По умолчанию: 1; Версия для контроля конкурентных изменений.</td></tr>
<tr><td class="attribute-name">created_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время создания.</td></tr>
<tr><td class="attribute-name">updated_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время последнего изменения.</td></tr>
<tr><td class="attribute-name" rowspan="10">template_revisions</td><td class="attribute-name">id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Идентификатор записи.</td></tr>
<tr><td class="attribute-name">template_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">revision</td><td class="attribute-type">integer</td><td class="attribute-description">Обязательно; Номер редакции.</td></tr>
<tr><td class="attribute-name">channel</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Канал доставки.</td></tr>
<tr><td class="attribute-name">locale</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Локаль представления.</td></tr>
<tr><td class="attribute-name">subject_template</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; Шаблон темы.</td></tr>
<tr><td class="attribute-name">body_template</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Шаблон текста.</td></tr>
<tr><td class="attribute-name">status</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Опубликованная редакция неизменяема</td></tr>
<tr><td class="attribute-name">version</td><td class="attribute-type">bigint</td><td class="attribute-description">Обязательно; По умолчанию: 1; Версия для контроля конкурентных изменений.</td></tr>
<tr><td class="attribute-name">created_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время создания.</td></tr>
<tr><td class="attribute-name" rowspan="17">delivery_jobs</td><td class="attribute-name">create_request_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Необязательно / не указано; UNIQUE; UUID операции создания; уникален в пределах этой таблицы</td></tr>
<tr><td class="attribute-name">create_request_hash</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; Хеш канонического запроса создания</td></tr>
<tr><td class="attribute-name">id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Идентификатор записи.</td></tr>
<tr><td class="attribute-name">recipient_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; Внешний Identity ID</td></tr>
<tr><td class="attribute-name">template_revision_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">rendered_subject</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; Фиксируется перед первой отправкой; содержит чувствительные данные</td></tr>
<tr><td class="attribute-name">rendered_body</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; Фиксируется перед первой отправкой; повтор использует тот же снимок</td></tr>
<tr><td class="attribute-name">template_payload</td><td class="attribute-type">jsonb</td><td class="attribute-description">Обязательно; Данные рендеринга; минимум персональных данных</td></tr>
<tr><td class="attribute-name">status</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Состояние записи; допустимые переходы описаны в проекте сервиса.</td></tr>
<tr><td class="attribute-name">scheduled_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время соответствующего действия или состояния.</td></tr>
<tr><td class="attribute-name">next_attempt_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Необязательно / не указано; Время следующей попытки.</td></tr>
<tr><td class="attribute-name">attempt_count</td><td class="attribute-type">integer</td><td class="attribute-description">Обязательно; По умолчанию: 0; Число попыток.</td></tr>
<tr><td class="attribute-name">lease_token</td><td class="attribute-type">uuid</td><td class="attribute-description">Необязательно / не указано; Токен захвата работы.</td></tr>
<tr><td class="attribute-name">lease_until</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Необязательно / не указано; Окончание срока захвата.</td></tr>
<tr><td class="attribute-name">version</td><td class="attribute-type">bigint</td><td class="attribute-description">Обязательно; По умолчанию: 1; Версия для контроля конкурентных изменений.</td></tr>
<tr><td class="attribute-name">created_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время создания.</td></tr>
<tr><td class="attribute-name">updated_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время последнего изменения.</td></tr>
<tr><td class="attribute-name" rowspan="9">delivery_attempts</td><td class="attribute-name">id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Идентификатор записи.</td></tr>
<tr><td class="attribute-name">job_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">ordinal</td><td class="attribute-type">integer</td><td class="attribute-description">Обязательно; Порядковый номер внутри родительского контекста.</td></tr>
<tr><td class="attribute-name">lease_token</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; Токен захвата работы.</td></tr>
<tr><td class="attribute-name">status</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Состояние записи; допустимые переходы описаны в проекте сервиса.</td></tr>
<tr><td class="attribute-name">provider_message_id</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">error_class</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; Классификация ошибки.</td></tr>
<tr><td class="attribute-name">started_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время начала.</td></tr>
<tr><td class="attribute-name">finished_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Необязательно / не указано; Время завершения.</td></tr>
<tr><td class="attribute-name" rowspan="6">in_app_notifications</td><td class="attribute-name">id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Идентификатор записи.</td></tr>
<tr><td class="attribute-name">delivery_job_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; FK, UNIQUE; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">recipient_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; FK; Внешний Identity ID</td></tr>
<tr><td class="attribute-name">title</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; Название.</td></tr>
<tr><td class="attribute-name">body</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Текст содержимого.</td></tr>
<tr><td class="attribute-name">created_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время создания.</td></tr>
<tr><td class="attribute-name" rowspan="4">processed_events</td><td class="attribute-name">consumer</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; PK; Идентификатор потребителя события.</td></tr>
<tr><td class="attribute-name">source_event_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">outcome</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Результат обработки.</td></tr>
<tr><td class="attribute-name">processed_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время соответствующего действия или состояния.</td></tr>
</tbody></table></div></div>

## Ограничения данных

- Ключ шаблона и локаль не пусты; номер редакции положителен. Канал выбирается из IN_APP, EMAIL, PUSH и SMS.
- Число попыток доставки неотрицательно. Для обрабатываемой задачи обязательны данные временного захвата; время окончания попытки не предшествует её началу.

- Доставка уведомления предназначена одному получателю и одному каналу; попыток доставки может быть несколько.

