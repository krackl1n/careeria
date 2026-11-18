# База данных сервиса Hiring

База данных Hiring хранит отклики кандидатов, этапы найма и историю переходов, сведения об интервью, его участниках и отзывах, а также предложения о работе. Эти данные обеспечивают ведение процесса найма от подачи отклика до итогового решения.

## ER-диаграмма



![ER-диаграмма Hiring](erd.svg)

<p align="center">Рисунок - Модель данных сервиса Hiring</p>

[Открыть SVG полностью](erd.svg)

## Сущности

<p align="right">Таблица - Сущности сервиса Hiring</p>

| Таблица | Псевдоним | Описание | Особенности |
|---|---|---|---|
| applications | Отклик | Отклик кандидата на вакансию и текущий этап найма. | PK: id |
| stage_transitions | Переход этапа | История изменения этапа отклика. | PK: id |
| interviews | Интервью | Назначенное собеседование и его состояние. | PK: id |
| interview_participants | Участник интервью | Участие пользователя в конкретном интервью. | PK: interview_id, identity_id |
| interview_feedback | Отзыв об интервью | Оценка и комментарий участника интервью. | PK: interview_id, author_id |
| job_offers | Предложение о работе | Условия, срок и состояние предложения кандидату. | PK: id |
| processed_events | Обработанное событие | Запись о результате обработки входящего события потребителем. | PK: consumer, event_id |
| outbox_messages | Исходящее событие | Сообщение transactional outbox для последующей публикации. | PK: id |

## Связи

<p align="right">Таблица - Связи сущностей сервиса Hiring</p>

| Родительская сущность и ключ | Дочерняя сущность и FK | Дочерних записей на одну родительскую | Родительских записей на одну дочернюю | Комментарий |
|---|---|---|---|---|
| applications (id) | stage_transitions (application_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: не задано. |
| applications (id) | interviews (application_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: не задано. |
| interviews (id) | interview_participants (interview_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: не задано. |
| interview_participants (interview_id, identity_id) | interview_feedback (interview_id, author_id) | 0..1 | 1 | Не более одной дочерней записи; ссылка обязательна. Удаление родителя: не задано. |
| applications (id) | job_offers (application_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: не задано. |

## Атрибуты

<p align="right">Таблица - Атрибуты сущностей сервиса Hiring</p>

<div class="md-typeset__scrollwrap attribute-dictionary"><div class="md-typeset__table"><table><thead><tr><th>Сущность</th><th>Поле</th><th>Тип</th><th>Описание</th></tr></thead><tbody>
<tr><td class="attribute-name" rowspan="9">applications</td><td class="attribute-name">create_request_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Необязательно / не указано; UNIQUE; UUID операции создания; уникален в пределах этой таблицы</td></tr>
<tr><td class="attribute-name">create_request_hash</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; Хеш канонического запроса создания</td></tr>
<tr><td class="attribute-name">id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Идентификатор записи.</td></tr>
<tr><td class="attribute-name">vacancy_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; Внешний Vacancy ID</td></tr>
<tr><td class="attribute-name">candidate_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; Внешний Candidate ID</td></tr>
<tr><td class="attribute-name">current_stage</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Текущий этап процесса.</td></tr>
<tr><td class="attribute-name">version</td><td class="attribute-type">bigint</td><td class="attribute-description">Обязательно; Версия для контроля конкурентных изменений.</td></tr>
<tr><td class="attribute-name">created_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время создания.</td></tr>
<tr><td class="attribute-name">updated_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время последнего изменения.</td></tr>
<tr><td class="attribute-name" rowspan="8">stage_transitions</td><td class="attribute-name">id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Идентификатор записи.</td></tr>
<tr><td class="attribute-name">application_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; FK; Идентификатор отклика.</td></tr>
<tr><td class="attribute-name">ordinal</td><td class="attribute-type">bigint</td><td class="attribute-description">Обязательно; Порядковый номер внутри родительского контекста.</td></tr>
<tr><td class="attribute-name">from_stage</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; NULL при начальном переходе</td></tr>
<tr><td class="attribute-name">to_stage</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Новый этап процесса.</td></tr>
<tr><td class="attribute-name">actor_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; Внешний инициатор или сервисная identity</td></tr>
<tr><td class="attribute-name">reason</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; Причина действия или перехода.</td></tr>
<tr><td class="attribute-name">occurred_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время события.</td></tr>
<tr><td class="attribute-name" rowspan="13">interviews</td><td class="attribute-name">create_request_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Необязательно / не указано; UNIQUE; UUID операции создания; уникален в пределах этой таблицы</td></tr>
<tr><td class="attribute-name">create_request_hash</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; Хеш канонического запроса создания</td></tr>
<tr><td class="attribute-name">id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Идентификатор записи.</td></tr>
<tr><td class="attribute-name">application_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; FK; Идентификатор отклика.</td></tr>
<tr><td class="attribute-name">interview_type</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Тип собеседования.</td></tr>
<tr><td class="attribute-name">meeting_format</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Формат проведения встречи.</td></tr>
<tr><td class="attribute-name">status</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Состояние записи; допустимые переходы описаны в проекте сервиса.</td></tr>
<tr><td class="attribute-name">scheduled_start_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Запланированное время начала.</td></tr>
<tr><td class="attribute-name">scheduled_end_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Запланированное время окончания.</td></tr>
<tr><td class="attribute-name">meeting_room_url</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; Состав и защита данных комнаты требуют решения</td></tr>
<tr><td class="attribute-name">version</td><td class="attribute-type">bigint</td><td class="attribute-description">Обязательно; Версия для контроля конкурентных изменений.</td></tr>
<tr><td class="attribute-name">created_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время создания.</td></tr>
<tr><td class="attribute-name">updated_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время последнего изменения.</td></tr>
<tr><td class="attribute-name" rowspan="3">interview_participants</td><td class="attribute-name">interview_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK, FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">identity_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Внешний Identity ID</td></tr>
<tr><td class="attribute-name">role</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Роль в контексте записи.</td></tr>
<tr><td class="attribute-name" rowspan="5">interview_feedback</td><td class="attribute-name">interview_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK, FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">author_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK, FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса.</td></tr>
<tr><td class="attribute-name">score</td><td class="attribute-type">integer</td><td class="attribute-description">Обязательно; Оценка.</td></tr>
<tr><td class="attribute-name">notes</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Примечания.</td></tr>
<tr><td class="attribute-name">submitted_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время отправки.</td></tr>
<tr><td class="attribute-name" rowspan="12">job_offers</td><td class="attribute-name">create_request_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Необязательно / не указано; UNIQUE; UUID операции создания; уникален в пределах этой таблицы</td></tr>
<tr><td class="attribute-name">create_request_hash</td><td class="attribute-type">text</td><td class="attribute-description">Необязательно / не указано; Хеш канонического запроса создания</td></tr>
<tr><td class="attribute-name">id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Идентификатор записи.</td></tr>
<tr><td class="attribute-name">application_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; FK; Идентификатор отклика.</td></tr>
<tr><td class="attribute-name">compensation</td><td class="attribute-type">bigint</td><td class="attribute-description">Обязательно; Единица и период суммы требуют согласования</td></tr>
<tr><td class="attribute-name">currency</td><td class="attribute-type">char(3)</td><td class="attribute-description">Обязательно; Код валюты.</td></tr>
<tr><td class="attribute-name">grade</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Уровень или грейд предложения.</td></tr>
<tr><td class="attribute-name">status</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Состояние записи; допустимые переходы описаны в проекте сервиса.</td></tr>
<tr><td class="attribute-name">expires_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время истечения.</td></tr>
<tr><td class="attribute-name">version</td><td class="attribute-type">bigint</td><td class="attribute-description">Обязательно; Версия для контроля конкурентных изменений.</td></tr>
<tr><td class="attribute-name">created_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время создания.</td></tr>
<tr><td class="attribute-name">updated_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время последнего изменения.</td></tr>
<tr><td class="attribute-name" rowspan="4">processed_events</td><td class="attribute-name">consumer</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; PK; Идентификатор потребителя события.</td></tr>
<tr><td class="attribute-name">event_id</td><td class="attribute-type">uuid</td><td class="attribute-description">Обязательно; PK; Идентификатор события.</td></tr>
<tr><td class="attribute-name">outcome</td><td class="attribute-type">text</td><td class="attribute-description">Обязательно; Результат обработки.</td></tr>
<tr><td class="attribute-name">processed_at</td><td class="attribute-type">timestamptz</td><td class="attribute-description">Обязательно; Время соответствующего действия или состояния.</td></tr>
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

- Отклик и интервью могут находиться только в состояниях, определённых их жизненным циклом. Порядковый номер перехода положителен.
- Время завершения интервью позже времени начала. Оценка отзыва находится в диапазоне от 1 до 5.
- Для предложения о работе сумма компенсации положительна, уровень должности не пуст; состояние предложения выбирается из предусмотренных состояний.

- Отклик относится к кандидату и вакансии; допустимые переходы этапов ограничены жизненным циклом найма. Проверка актуальности данных других сервисов требует отдельного контракта.
- Отзыв об интервью должен принадлежать участнику этого интервью; право конкретной роли оставлять отзыв проверяется отдельно.
- Решение по предложению о работе проверяет его состояние и срок на момент обработки.

