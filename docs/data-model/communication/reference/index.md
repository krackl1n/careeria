# База данных сервиса Communication

База данных Communication хранит диалоги, их участников, сообщения, вложения, реакции и отметки о прочтении. Модель поддерживает личное общение и диалоги, связанные с процессом найма, а также управление состоянием сообщений и участников.

## ER-диаграмма



![ER-диаграмма Communication](erd.svg)

<p align="center">Рисунок - Модель данных сервиса Communication</p>

[Открыть SVG полностью](erd.svg)

## Сущности

<p align="right">Таблица - Сущности сервиса Communication</p>

| Таблица | Псевдоним | Описание | Особенности |
|---|---|---|---|
| conversations | Диалог | Состояние диалога и порядок сообщений. | PK: id |
| direct_conversation_pairs | Пара участников | Каноническая пара пользователей личного диалога. | PK: conversation_id |
| conversation_participants | Участник диалога | Членство и настройки пользователя в диалоге. | PK: conversation_id, identity_id |
| messages | Сообщение | Содержимое сообщения и его положение в диалоге. | PK: id |
| message_attachments | Вложение сообщения | Ссылка на файл и метаданные вложения. | PK: id |
| conversation_read_receipts | Отметка прочтения | Позиция прочтения участника диалога. | PK: conversation_id, identity_id |
| message_reactions | Реакция на сообщение | Реакция участника с выбранным кодом. | PK: message_id, identity_id, emoji_code |
| attachment_cleanup_jobs | Очистка вложения | Задание на освобождение файловой привязки. | PK: id |
| outbox_messages | Исходящее событие | Сообщение transactional outbox для последующей публикации. | PK: id |

## Связи

<p align="right">Таблица - Связи сущностей сервиса Communication</p>

| Родительская сущность и ключ | Дочерняя сущность и FK | Дочерних записей на одну родительскую | Родительских записей на одну дочернюю | Комментарий |
|---|---|---|---|---|
| conversations (id) | direct_conversation_pairs (conversation_id) | 0..1 | 1 | Не более одной дочерней записи; ссылка обязательна. Удаление родителя: не задано. |
| conversations (id) | conversation_participants (conversation_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: не задано. |
| conversations (id) | messages (conversation_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: не задано. |
| conversation_participants (conversation_id, identity_id) | messages (conversation_id, sender_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: не задано. |
| messages (conversation_id, id) | messages (conversation_id, reply_to_message_id) | 0..N | 0..1 | Безусловная уникальность FK не указана; ссылка может отсутствовать. Удаление родителя: не задано. |
| messages (id) | message_attachments (message_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: не задано. |
| conversation_participants (conversation_id, identity_id) | conversation_read_receipts (conversation_id, identity_id) | 0..1 | 1 | Не более одной дочерней записи; ссылка обязательна. Удаление родителя: не задано. |
| messages (conversation_id, sequence) | conversation_read_receipts (conversation_id, last_read_sequence) | 0..N | 0..1 | Безусловная уникальность FK не указана; ссылка может отсутствовать. Удаление родителя: не задано. |
| messages (id) | message_reactions (message_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: не задано. |

## Атрибуты

<p align="right">Таблица - Атрибуты сущностей сервиса Communication</p>

<div class="md-typeset__scrollwrap attribute-dictionary"><div class="md-typeset__table"><table><thead><tr><th>Сущность</th><th>Поле</th><th>Тип</th><th>Описание</th></tr></thead><tbody>
<tr><td class="attribute-name" rowspan="10">conversations</td><td class="attribute-name">create_request_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Необязательно / не указано; UNIQUE; UUID операции создания; уникален в пределах этой таблицы</td></tr>
<tr><td class="attribute-name">create_request_hash</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; Хеш канонического запроса создания</td></tr>
<tr><td class="attribute-name">id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Идентификатор записи.</td></tr>
<tr><td class="attribute-name">type</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Тип объекта согласно модели сервиса.</td></tr>
<tr><td class="attribute-name">context_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Необязательно / не указано; Внешний ID отклика или интервью</td></tr>
<tr><td class="attribute-name">status</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Состояние записи; допустимые переходы описаны в проекте сервиса.</td></tr>
<tr><td class="attribute-name">next_sequence</td><td class="attribute-type">bigint</td><td class="attribute-description">Обязательно; По умолчанию: 1; Следующий номер сообщения в диалоге.</td></tr>
<tr><td class="attribute-name">version</td><td class="attribute-type">bigint</td><td class="attribute-description">Обязательно; По умолчанию: 1; Версия для контроля конкурентных изменений.</td></tr>
<tr><td class="attribute-name">created_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время создания.</td></tr>
<tr><td class="attribute-name">updated_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время последнего изменения.</td></tr>
<tr><td class="attribute-name" rowspan="3">direct_conversation_pairs</td><td class="attribute-name">conversation_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK, FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">identity_low</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; Внешний Identity ID</td></tr>
<tr><td class="attribute-name">identity_high</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; Внешний Identity ID</td></tr>
<tr><td class="attribute-name" rowspan="8">conversation_participants</td><td class="attribute-name">conversation_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK, FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">identity_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Внешний Identity ID</td></tr>
<tr><td class="attribute-name">role</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Роль в контексте записи.</td></tr>
<tr><td class="attribute-name">status</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Состояние записи; допустимые переходы описаны в проекте сервиса.</td></tr>
<tr><td class="attribute-name">joined_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время соответствующего действия или состояния.</td></tr>
<tr><td class="attribute-name">removed_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Необязательно / не указано; Время соответствующего действия или состояния.</td></tr>
<tr><td class="attribute-name">archived_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Необязательно / не указано; Персональное скрытие из активного списка</td></tr>
<tr><td class="attribute-name">is_muted</td><td class="attribute-type">boolean</td><td class="attribute-description">Обязательно; По умолчанию: false; Настройка отключения уведомлений.</td></tr>
<tr><td class="attribute-name" rowspan="14">messages</td><td class="attribute-name">create_request_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Необязательно / не указано; UNIQUE; UUID операции создания; уникален в пределах этой таблицы</td></tr>
<tr><td class="attribute-name">create_request_hash</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; Хеш канонического запроса создания</td></tr>
<tr><td class="attribute-name">id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Идентификатор записи.</td></tr>
<tr><td class="attribute-name">conversation_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">sequence</td><td class="attribute-type">bigint</td><td class="attribute-description">Обязательно; Порядковый номер внутри родительского контекста.</td></tr>
<tr><td class="attribute-name">sender_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; FK; Внешний Identity ID</td></tr>
<tr><td class="attribute-name">reply_to_message_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Необязательно / не указано; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">content</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; Содержимое, допустимое для этого объекта.</td></tr>
<tr><td class="attribute-name">status</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Состояние записи; допустимые переходы описаны в проекте сервиса.</td></tr>
<tr><td class="attribute-name">version</td><td class="attribute-type">bigint</td><td class="attribute-description">Обязательно; По умолчанию: 1; Версия для контроля конкурентных изменений.</td></tr>
<tr><td class="attribute-name">created_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время создания.</td></tr>
<tr><td class="attribute-name">updated_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время последнего изменения.</td></tr>
<tr><td class="attribute-name">edited_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Необязательно / не указано; Время соответствующего действия или состояния.</td></tr>
<tr><td class="attribute-name">deleted_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Необязательно / не указано; Время удаления.</td></tr>
<tr><td class="attribute-name" rowspan="8">message_attachments</td><td class="attribute-name">id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Идентификатор записи.</td></tr>
<tr><td class="attribute-name">message_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">file_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; Внешний File ID</td></tr>
<tr><td class="attribute-name">binding_operation_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; UNIQUE; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">file_name</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Отображаемое имя файла.</td></tr>
<tr><td class="attribute-name">size_bytes</td><td class="attribute-type">bigint</td><td class="attribute-description">Обязательно; Размер файла в байтах.</td></tr>
<tr><td class="attribute-name">mime_type</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; MIME-тип файла.</td></tr>
<tr><td class="attribute-name">created_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время создания.</td></tr>
<tr><td class="attribute-name" rowspan="4">conversation_read_receipts</td><td class="attribute-name">conversation_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK, FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">identity_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK, FK; Идентификатор учётной записи Identity.</td></tr>
<tr><td class="attribute-name">last_read_sequence</td><td class="attribute-type">bigint</td><td class="attribute-description">Необязательно / не указано; FK; NULL до первого прочтения</td></tr>
<tr><td class="attribute-name">updated_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время последнего изменения.</td></tr>
<tr><td class="attribute-name" rowspan="4">message_reactions</td><td class="attribute-name">message_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK, FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">identity_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Внешний Identity ID; активное членство проверяется сервисом</td></tr>
<tr><td class="attribute-name">emoji_code</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; PK; Код реакции.</td></tr>
<tr><td class="attribute-name">created_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время создания.</td></tr>
<tr><td class="attribute-name" rowspan="10">attachment_cleanup_jobs</td><td class="attribute-name">id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Идентификатор записи.</td></tr>
<tr><td class="attribute-name">file_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; Идентификатор файла.</td></tr>
<tr><td class="attribute-name">binding_operation_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; UNIQUE; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">status</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Состояние записи; допустимые переходы описаны в проекте сервиса.</td></tr>
<tr><td class="attribute-name">attempts</td><td class="attribute-type">integer</td><td class="attribute-description">Обязательно; По умолчанию: 0; Количество попыток.</td></tr>
<tr><td class="attribute-name">next_attempt_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Необязательно / не указано; Время следующей попытки.</td></tr>
<tr><td class="attribute-name">lease_token</td><td class="attribute-type">uuid</td><td class="attribute-description">Необязательно / не указано; Токен захвата работы.</td></tr>
<tr><td class="attribute-name">lease_until</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Необязательно / не указано; Окончание срока захвата.</td></tr>
<tr><td class="attribute-name">created_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время создания.</td></tr>
<tr><td class="attribute-name">updated_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время последнего изменения.</td></tr>
<tr><td class="attribute-name" rowspan="18">outbox_messages</td><td class="attribute-name">id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Идентификатор записи.</td></tr>
<tr><td class="attribute-name">event_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; UNIQUE; Идентификатор события.</td></tr>
<tr><td class="attribute-name">aggregate_type</td><td class="attribute-type">varchar(255)</td><td class="attribute-description">Обязательно; Тип агрегата — источника события.</td></tr>
<tr><td class="attribute-name">aggregate_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">aggregate_version</td><td class="attribute-type">bigint</td><td class="attribute-description">Необязательно / не указано; Версия агрегата — источника события.</td></tr>
<tr><td class="attribute-name">event_type</td><td class="attribute-type">varchar(255)</td><td class="attribute-description">Обязательно; Тип события.</td></tr>
<tr><td class="attribute-name">payload</td><td class="attribute-type">jsonb</td><td class="attribute-description">Обязательно; Содержимое события или операции.</td></tr>
<tr><td class="attribute-name">headers</td><td class="attribute-type">jsonb</td><td class="attribute-description">Обязательно; По умолчанию: {}; Заголовки события.</td></tr>
<tr><td class="attribute-name">partition_key</td><td class="attribute-type">varchar(255)</td><td class="attribute-description">Необязательно / не указано; Ключ распределения события.</td></tr>
<tr><td class="attribute-name">status</td><td class="attribute-type">varchar(32)</td><td class="attribute-description">Обязательно; По умолчанию: PENDING; Состояние записи; допустимые переходы описаны в проекте сервиса.</td></tr>
<tr><td class="attribute-name">attempts</td><td class="attribute-type">integer</td><td class="attribute-description">Обязательно; По умолчанию: 0; Количество попыток.</td></tr>
<tr><td class="attribute-name">occurred_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время события.</td></tr>
<tr><td class="attribute-name">processing_started_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Необязательно / не указано; Время соответствующего действия или состояния.</td></tr>
<tr><td class="attribute-name">published_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Необязательно / не указано; Время публикации.</td></tr>
<tr><td class="attribute-name">next_attempt_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Необязательно / не указано; Время следующей попытки.</td></tr>
<tr><td class="attribute-name">last_error</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; Последняя зафиксированная ошибка.</td></tr>
<tr><td class="attribute-name">created_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время создания.</td></tr>
<tr><td class="attribute-name">updated_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время последнего изменения.</td></tr>
</tbody></table></div></div>

## Ограничения данных

- Для личного диалога контекст не задаётся; для диалога по отклику или интервью контекст обязателен. Состояние и тип диалога выбираются из предусмотренных значений.
- Для удалённого участника фиксируется время удаления. Удалённое сообщение не содержит текст и имеет время удаления.
- Сообщение не может ссылаться на само себя; размер вложения положителен, код реакции не пуст.

- Личный диалог связывает двух разных пользователей; пара пользователей не может иметь более одного такого диалога в предлагаемой модели.
- Сообщение отправляет участник активного диалога; ответ должен ссылаться на сообщение того же диалога.
- Указатель прочтения участника может перемещаться только вперёд и должен относиться к сообщению того же диалога.

