# База данных сервиса Identity

Источник: существующая DBML `services/identity/db/schema.dbml`; миграции Identity имеют приоритет. Смысл сценариев и незакрытые решения — в [описании сервиса](../../../services/identity/index.md).

## ER-диаграмма

Таблицы показаны со всеми полями из DBML. PK — первичный ключ, FK — внешний ключ, звёздочка — обязательное поле. Связи обозначены нотацией «воронья лапка»: круг — необязательная связь, черта — один, лапка — многие. Линии подходят к строкам полей; для составного FK линия подходит к последнему полю, а все поля перечислены ниже. Кратность выведена из FK, NOT NULL и уникальных ключей DBML; дополнительные SQL-ограничения смотрите в миграциях или проекте ограничений. Внешние идентификаторы других сервисов не соединены межбазовыми FK.

Identity разделена по существующим группам DBML; общие таблицы повторяются только в тех разделах, где они участвуют в связях.

### Учётные записи и контакты

![ER-диаграмма Identity: Учётные записи и контакты](erd-identity_core.svg)

[Открыть SVG: Учётные записи и контакты](erd-identity_core.svg)

### Многофакторная аутентификация и восстановление

![ER-диаграмма Identity: Многофакторная аутентификация и восстановление](erd-mfa_and_recovery.svg)

[Открыть SVG: Многофакторная аутентификация и восстановление](erd-mfa_and_recovery.svg)

### Проверка контактов и сброс пароля

![ER-диаграмма Identity: Проверка контактов и сброс пароля](erd-verification_challenges.svg)

[Открыть SVG: Проверка контактов и сброс пароля](erd-verification_challenges.svg)

### OAuth-клиенты

![ER-диаграмма Identity: OAuth-клиенты](erd-oauth_config.svg)

[Открыть SVG: OAuth-клиенты](erd-oauth_config.svg)

### OAuth-согласия

![ER-диаграмма Identity: OAuth-согласия](erd-oauth_grants.svg)

[Открыть SVG: OAuth-согласия](erd-oauth_grants.svg)

### Сессии и токены

![ER-диаграмма Identity: Сессии и токены](erd-sessions_and_tokens.svg)

[Открыть SVG: Сессии и токены](erd-sessions_and_tokens.svg)

### Сервисные учётные записи

![ER-диаграмма Identity: Сервисные учётные записи](erd-service_accounts.svg)

[Открыть SVG: Сервисные учётные записи](erd-service_accounts.svg)

### События безопасности

![ER-диаграмма Identity: События безопасности](erd-security_audit.svg)

[Открыть SVG: События безопасности](erd-security_audit.svg)

### Outbox

![ER-диаграмма Identity: Outbox](erd-messaging.svg)

[Открыть SVG: Outbox](erd-messaging.svg)

## Сущности

Здесь сущность — таблица локальной базы, включая технические таблицы. Название на русском — поясняющий псевдоним; физическое имя приведено в первом столбце.

| Таблица | Псевдоним | Описание | Особенности |
|---|---|---|---|
| identities | Учётная запись | Идентичность пользователя и её состояние. | PK: id; CHECK: deleted status requires deleted_at; updated_at >= created_at. |
| identity_profiles | Профиль учётной записи | Основные персональные данные учётной записи. | PK: identity_id; One-to-one identity profile. Names are trimmed and must not be blank. The application increments version on every profile change. |
| identity_emails | Электронный адрес | Адрес учётной записи и состояние подтверждения. | PK: id; Email and identity_id are immutable. Email is normalized to lowercase; primary contact cannot be deleted. |
| identity_phones | Номер телефона | Телефон учётной записи и состояние подтверждения. | PK: id; Phone and identity_id are immutable. Phone uses E.164 format; primary contact cannot be deleted. |
| external_identity_accounts | Внешняя учётная запись | Связь идентичности с субъектом внешнего провайдера. | PK: id; identity_id, provider, subject and linked_at are immutable. |
| identity_password_credentials | Парольные данные | Хеш пароля и параметры его использования. | PK: identity_id |
| identity_password_auth_state | Состояние парольной защиты | Счётчики неудачных попыток и блокировка входа. | PK: identity_id; locked_until and lock_reason are set or cleared together. |
| mfa_factors | Фактор аутентификации | Общие сведения и состояние дополнительного фактора. | PK: id; identity_id, factor_type and created_at are immutable. |
| totp_factors | Фактор TOTP | Параметры и секрет фактора одноразовых кодов. | PK: factor_id; Subtype of mfa_factors; factor_type must be totp. |
| webauthn_credentials | Учётные данные WebAuthn | Параметры зарегистрированного аутентификатора. | PK: factor_id; Subtype of mfa_factors; factor_type must be webauthn. |
| recovery_code_sets | Набор кодов восстановления | Состояние набора резервных кодов. | PK: id; identity_id and created_at are immutable. |
| recovery_codes | Код восстановления | Хеш и состояние отдельного резервного кода. | PK: id; set_id, code_hash and created_at are immutable. |
| email_verification_challenges | Проверка электронной почты | Запрос подтверждения адреса электронной почты. | PK: id |
| phone_verification_challenges | Проверка телефона | Запрос подтверждения номера телефона. | PK: id |
| password_reset_challenges | Сброс пароля | Запрос на восстановление доступа по паролю. | PK: id |
| oauth_clients | OAuth-клиент | Конфигурация приложения, использующего OAuth. | PK: id; public clients require PKCE. client_id, client_type, auth method and created_at are immutable. |
| oauth_client_redirect_uris | Адрес перенаправления клиента | Разрешённый redirect URI OAuth-клиента. | PK: id |
| oauth_client_post_logout_redirect_uris | Адрес после выхода | Разрешённый адрес перехода после выхода из системы. | PK: id |
| oauth_client_grant_types | Тип выдачи OAuth | Разрешённый клиенту тип авторизационного grant. | PK: oauth_client_id, grant_type |
| oauth_client_scopes | Область доступа клиента | Область доступа, которую клиент может запрашивать. | PK: oauth_client_id, scope; Maximum scopes this OAuth client is allowed to request; this is not a user consent. |
| oauth_client_credentials | Учётные данные OAuth-клиента | Секрет и состояние аутентификации клиента. | PK: id; Only confidential clients can have credentials. Secret and protocol identity are immutable. |
| oauth_authorization_grants | Согласие на доступ | Запомненное разрешение между пользователем и клиентом. | PK: id; Current remembered authorization between one identity and one OAuth client. This is separate from client configuration and from a single authorization transaction. |
| oauth_authorization_grant_scopes | Область согласия | Область доступа, включённая в разрешение пользователя. | PK: grant_id, scope; Scopes remembered for the identity-client authorization grant. They must remain a subset of oauth_client_scopes. |
| sessions | Сессия | Состояние аутентифицированной сессии и сроки её действия. | PK: id; Authentication identity, client binding and absolute expiration are immutable. |
| session_scopes | Область доступа сессии | Снимок области доступа, предоставленной сессии. | PK: session_id, scope; Snapshot of scopes granted when the session was created. |
| refresh_token_families | Семейство refresh-токенов | Объединение токенов для ротации и отзыва. | PK: id; Session binding, creation time and absolute expiration are immutable. |
| refresh_tokens | Refresh-токен | Хеш токена и сведения о его ротации. | PK: id; Parent must belong to the same family. Token identity and issuance data are immutable. |
| signing_keys | Ключ подписи | Публичные параметры и ссылка на закрытый ключ подписи. | PK: id; public_jwk rejects private JWK parameters; private material is referenced externally. |
| service_accounts | Сервисная учётная запись | Идентичность вызывающего сервиса. | PK: id; organization_id, account_key and created_at are immutable. |
| service_account_credentials | Учётные данные сервиса | Секрет и состояние доступа сервисной учётной записи. | PK: id; Credential binding, type, secret hash and created_at are immutable. |
| security_events | Событие безопасности | Журнал события аутентификации или безопасности. | PK: id; Actor type determines whether identity_id or service_account_id must be present. |
| outbox_messages | Исходящее событие | Сообщение transactional outbox для последующей публикации. | PK: id; Transactional outbox. Payload and headers are JSON objects; processing state controls retries and publication. |

## Связи

Одна строка соответствует одному локальному FK. Кратность указана в двух направлениях: сколько дочерних записей может относиться к одной родительской и сколько родительских — к одной дочерней. Минимум дочерних записей равен нулю: FK не требует наличия дочерней записи. Для составного nullable FK значение 0..1 означает, что ссылка может отсутствовать; частично заполненный ключ при стандартном MATCH SIMPLE также не проверяется. Указана кратность по DBML без учёта частичных уникальных индексов: 0..N — верхняя оценка, которую дополнительные ограничения могут сузить. В Identity приоритет имеют миграции.

| Родительская сущность и ключ | Дочерняя сущность и FK | Дочерних на родителя | Родителей на дочернюю | Комментарий |
|---|---|---|---|---|
| identities (id) | identity_emails (identity_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| identities (id) | identity_profiles (identity_id) | 0..1 | 1 | Не более одной дочерней записи; ссылка обязательна. Удаление родителя: cascade. |
| identities (id) | identity_phones (identity_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| identities (id) | external_identity_accounts (identity_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| identities (id) | identity_password_credentials (identity_id) | 0..1 | 1 | Не более одной дочерней записи; ссылка обязательна. Удаление родителя: cascade. |
| identities (id) | identity_password_auth_state (identity_id) | 0..1 | 1 | Не более одной дочерней записи; ссылка обязательна. Удаление родителя: cascade. |
| identities (id) | mfa_factors (identity_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| mfa_factors (id, factor_type) | totp_factors (factor_id, factor_type) | 0..1 | 1 | Не более одной дочерней записи; ссылка обязательна. Удаление родителя: cascade. |
| mfa_factors (id, factor_type) | webauthn_credentials (factor_id, factor_type) | 0..1 | 1 | Не более одной дочерней записи; ссылка обязательна. Удаление родителя: cascade. |
| identities (id) | recovery_code_sets (identity_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| recovery_code_sets (id) | recovery_codes (set_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| identity_emails (id) | email_verification_challenges (email_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| identity_phones (id) | phone_verification_challenges (phone_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| identities (id) | password_reset_challenges (identity_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| identities (id) | oauth_clients (created_by_identity_id) | 0..N | 0..1 | Безусловная уникальность FK не указана; ссылка может отсутствовать. Удаление родителя: set null. |
| oauth_clients (id) | oauth_client_redirect_uris (oauth_client_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| oauth_clients (id) | oauth_client_post_logout_redirect_uris (oauth_client_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| oauth_clients (id) | oauth_client_grant_types (oauth_client_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| oauth_clients (id) | oauth_client_scopes (oauth_client_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| oauth_clients (id, client_type) | oauth_client_credentials (oauth_client_id, client_type) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| identities (id) | oauth_client_credentials (created_by_identity_id) | 0..N | 0..1 | Безусловная уникальность FK не указана; ссылка может отсутствовать. Удаление родителя: set null. |
| identities (id) | oauth_authorization_grants (identity_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| oauth_clients (id) | oauth_authorization_grants (oauth_client_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| oauth_authorization_grants (id) | oauth_authorization_grant_scopes (grant_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| identities (id) | sessions (identity_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| oauth_clients (id) | sessions (oauth_client_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: restrict. |
| sessions (id) | session_scopes (session_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| sessions (id) | refresh_token_families (session_id) | 0..1 | 1 | Не более одной дочерней записи; ссылка обязательна. Удаление родителя: cascade. |
| refresh_token_families (id) | refresh_tokens (family_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| refresh_tokens (id, family_id) | refresh_tokens (parent_token_id, family_id) | 0..N | 0..1 | Безусловная уникальность FK не указана; ссылка может отсутствовать. Удаление родителя: не указано в DBML. |
| identities (id) | service_accounts (created_by_identity_id) | 0..N | 0..1 | Безусловная уникальность FK не указана; ссылка может отсутствовать. Удаление родителя: set null. |
| service_accounts (id) | service_account_credentials (service_account_id) | 0..N | 1 | Безусловная уникальность FK не указана; ссылка обязательна. Удаление родителя: cascade. |
| identities (id) | service_account_credentials (created_by_identity_id) | 0..N | 0..1 | Безусловная уникальность FK не указана; ссылка может отсутствовать. Удаление родителя: set null. |
| identities (id) | security_events (identity_id) | 0..N | 0..1 | Безусловная уникальность FK не указана; ссылка может отсутствовать. Удаление родителя: set null. |
| service_accounts (id) | security_events (service_account_id) | 0..N | 0..1 | Безусловная уникальность FK не указана; ссылка может отсутствовать. Удаление родителя: set null. |
| sessions (id) | security_events (session_id) | 0..N | 0..1 | Безусловная уникальность FK не указана; ссылка может отсутствовать. Удаление родителя: set null. |
| oauth_clients (id) | security_events (oauth_client_id) | 0..N | 0..1 | Безусловная уникальность FK не указана; ссылка может отсутствовать. Удаление родителя: set null. |

## Атрибуты

В описании указаны обязательность по DBML, ключи, значение по умолчанию и смысл поля. «Необязательно / не указано» не исключает дополнительных SQL-ограничений. UNIQUE отмечен для отдельного поля; составные ключи и индексы перечислены под таблицей.

### identities

CHECK: deleted status requires deleted_at; updated_at >= created_at.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| status | public.identity_status | Обязательно; По умолчанию: pending; Состояние записи; допустимые переходы описаны в проекте сервиса. |
| created_at | timestamptz | Обязательно; Время создания. |
| updated_at | timestamptz | Обязательно; Время последнего изменения. |
| deleted_at | timestamptz | Необязательно / не указано; Время удаления. |

Составные ключи и индексы DBML: `INDEX (status)`; `INDEX (deleted_at) — WHERE deleted_at IS NOT NULL`.

### identity_profiles

One-to-one identity profile. Names are trimmed and must not be blank. The application increments version on every profile change.

| Поле | Тип | Описание |
|---|---|---|
| identity_id | uuid | Обязательно; PK, FK; Идентификатор учётной записи Identity. |
| family_name | text | Обязательно; Фамилия. |
| given_name | text | Обязательно; Имя. |
| middle_name | text | Необязательно / не указано; Отчество или второе имя. |
| source | public.identity_profile_source | Обязательно; По умолчанию: self_declared; Источник данных. |
| assurance_level | public.identity_assurance_level | Обязательно; По умолчанию: self_declared; Уровень подтверждения данных. |
| version | bigint | Обязательно; По умолчанию: 1; Версия для контроля конкурентных изменений. |
| created_at | timestamptz | Обязательно; Время создания. |
| updated_at | timestamptz | Обязательно; Время последнего изменения. |

### identity_emails

Email and identity_id are immutable. Email is normalized to lowercase; primary contact cannot be deleted.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| identity_id | uuid | Обязательно; FK; Идентификатор учётной записи Identity. |
| email | text | Обязательно; UNIQUE; Адрес электронной почты. |
| is_primary | boolean | Обязательно; По умолчанию: false; Признак основного значения. |
| verified_at | timestamptz | Необязательно / не указано; Время подтверждения. |
| created_at | timestamptz | Обязательно; Время создания. |
| updated_at | timestamptz | Обязательно; Время последнего изменения. |
| deleted_at | timestamptz | Необязательно / не указано; Время удаления. |

Составные ключи и индексы DBML: `UNIQUE (email)`; `INDEX (identity_id)`; `UNIQUE (identity_id) — WHERE is_primary AND deleted_at IS NULL`.

### identity_phones

Phone and identity_id are immutable. Phone uses E.164 format; primary contact cannot be deleted.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| identity_id | uuid | Обязательно; FK; Идентификатор учётной записи Identity. |
| phone | text | Обязательно; UNIQUE; Номер телефона. |
| is_primary | boolean | Обязательно; По умолчанию: false; Признак основного значения. |
| verified_at | timestamptz | Необязательно / не указано; Время подтверждения. |
| created_at | timestamptz | Обязательно; Время создания. |
| updated_at | timestamptz | Обязательно; Время последнего изменения. |
| deleted_at | timestamptz | Необязательно / не указано; Время удаления. |

Составные ключи и индексы DBML: `UNIQUE (phone)`; `INDEX (identity_id)`; `UNIQUE (identity_id) — WHERE is_primary AND deleted_at IS NULL`.

### external_identity_accounts

identity_id, provider, subject and linked_at are immutable.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| identity_id | uuid | Обязательно; FK; Идентификатор учётной записи Identity. |
| provider | text | Обязательно; Идентификатор провайдера. |
| subject | text | Обязательно; Идентификатор субъекта. |
| linked_at | timestamptz | Обязательно; Время соответствующего действия или состояния. |
| last_authenticated_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| disabled_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| deleted_at | timestamptz | Необязательно / не указано; Время удаления. |

Составные ключи и индексы DBML: `UNIQUE (provider, subject)`; `INDEX (identity_id)`.

### identity_password_credentials

| Поле | Тип | Описание |
|---|---|---|
| identity_id | uuid | Обязательно; PK, FK; Идентификатор учётной записи Identity. |
| password_hash | text | Обязательно; Хеш пароля. |
| must_change | boolean | Обязательно; По умолчанию: false; Признак обязательной смены. |
| password_updated_at | timestamptz | Обязательно; Время соответствующего действия или состояния. |
| expires_at | timestamptz | Необязательно / не указано; Время истечения. |
| compromised_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| created_at | timestamptz | Обязательно; Время создания. |
| updated_at | timestamptz | Обязательно; Время последнего изменения. |

### identity_password_auth_state

locked_until and lock_reason are set or cleared together.

| Поле | Тип | Описание |
|---|---|---|
| identity_id | uuid | Обязательно; PK, FK; Идентификатор учётной записи Identity. |
| failed_attempts | integer | Обязательно; По умолчанию: 0; Число неудачных попыток. |
| last_failed_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| locked_until | timestamptz | Необязательно / не указано; Время окончания блокировки. |
| lock_reason | text | Необязательно / не указано; Причина блокировки. |
| updated_at | timestamptz | Обязательно; Время последнего изменения. |

Составные ключи и индексы DBML: `INDEX (locked_until) — WHERE locked_until IS NOT NULL`.

### mfa_factors

identity_id, factor_type and created_at are immutable.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| identity_id | uuid | Обязательно; FK; Идентификатор учётной записи Identity. |
| factor_type | public.mfa_factor_type | Обязательно; Тип фактора аутентификации. |
| name | text | Необязательно / не указано; Название или имя. |
| is_primary | boolean | Обязательно; По умолчанию: false; Признак основного значения. |
| confirmed_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| created_at | timestamptz | Обязательно; Время создания. |
| updated_at | timestamptz | Обязательно; Время последнего изменения. |
| last_used_at | timestamptz | Необязательно / не указано; Время последнего использования. |
| disabled_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |

Составные ключи и индексы DBML: `UNIQUE (id, factor_type)`; `INDEX (identity_id)`; `UNIQUE (identity_id) — WHERE is_primary AND confirmed_at IS NOT NULL AND disabled_at IS NULL`; `UNIQUE (identity_id) — WHERE factor_type = 'totp' AND disabled_at IS NULL`.

### totp_factors

Subtype of mfa_factors; factor_type must be totp.

| Поле | Тип | Описание |
|---|---|---|
| factor_id | uuid | Обязательно; PK, FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| factor_type | public.mfa_factor_type | Обязательно; FK; По умолчанию: totp; Тип фактора аутентификации. |
| secret_encrypted | bytea | Обязательно; Зашифрованный секрет. |
| encryption_key_ref | text | Обязательно; Ссылка на ключ шифрования. |
| algorithm | public.totp_algorithm | Обязательно; По умолчанию: sha1; Идентификатор алгоритма. |
| digits | smallint | Обязательно; По умолчанию: 6; Количество цифр одноразового кода. |
| period_seconds | smallint | Обязательно; По умолчанию: 30; Продолжительность периода в секундах. |
| last_used_time_step | bigint | Необязательно / не указано; Последний использованный временной шаг. |
| updated_at | timestamptz | Обязательно; Время последнего изменения. |

### webauthn_credentials

Subtype of mfa_factors; factor_type must be webauthn.

| Поле | Тип | Описание |
|---|---|---|
| factor_id | uuid | Обязательно; PK, FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| factor_type | public.mfa_factor_type | Обязательно; FK; По умолчанию: webauthn; Тип фактора аутентификации. |
| credential_id | bytea | Обязательно; UNIQUE; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| public_key | bytea | Обязательно; Открытый ключ. |
| sign_count | bigint | Обязательно; По умолчанию: 0; Счётчик подписей. |
| aaguid | uuid | Необязательно / не указано; Идентификатор типа аутентификатора AAGUID. |
| transports | text[] | Обязательно; По умолчанию: {}; Способы взаимодействия с аутентификатором. |
| backup_eligible | boolean | Обязательно; По умолчанию: false; Признак возможности резервирования. |
| backup_state | boolean | Обязательно; По умолчанию: false; Признак состояния резервирования. |
| created_at | timestamptz | Обязательно; Время создания. |
| last_used_at | timestamptz | Необязательно / не указано; Время последнего использования. |

Составные ключи и индексы DBML: `UNIQUE (credential_id)`.

### recovery_code_sets

identity_id and created_at are immutable.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| identity_id | uuid | Обязательно; FK; Идентификатор учётной записи Identity. |
| created_at | timestamptz | Обязательно; Время создания. |
| expires_at | timestamptz | Необязательно / не указано; Время истечения. |
| disabled_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |

Составные ключи и индексы DBML: `INDEX (identity_id)`; `UNIQUE (identity_id) — WHERE disabled_at IS NULL`.

### recovery_codes

set_id, code_hash and created_at are immutable.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| set_id | uuid | Обязательно; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| code_hash | text | Обязательно; Хеш кода. |
| created_at | timestamptz | Обязательно; Время создания. |
| used_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |

Составные ключи и индексы DBML: `UNIQUE (set_id, code_hash)`; `INDEX (set_id) — WHERE used_at IS NULL`.

### email_verification_challenges

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| email_id | uuid | Обязательно; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| token_hash | text | Обязательно; UNIQUE; Хеш токена. |
| attempts | integer | Обязательно; По умолчанию: 0; Количество попыток. |
| max_attempts | integer | Обязательно; Максимально допустимое число попыток. |
| expires_at | timestamptz | Обязательно; Время истечения. |
| last_attempt_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| consumed_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| revoked_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| requested_from_ip | inet | Необязательно / не указано; IP-адрес источника запроса. |
| consumed_from_ip | inet | Необязательно / не указано; IP-адрес источника использования. |
| created_at | timestamptz | Обязательно; Время создания. |

Составные ключи и индексы DBML: `UNIQUE (token_hash)`; `INDEX (email_id, created_at)`; `INDEX (expires_at) — WHERE consumed_at IS NULL AND revoked_at IS NULL`.

### phone_verification_challenges

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| phone_id | uuid | Обязательно; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| token_hash | text | Обязательно; UNIQUE; Хеш токена. |
| attempts | integer | Обязательно; По умолчанию: 0; Количество попыток. |
| max_attempts | integer | Обязательно; Максимально допустимое число попыток. |
| expires_at | timestamptz | Обязательно; Время истечения. |
| last_attempt_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| consumed_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| revoked_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| requested_from_ip | inet | Необязательно / не указано; IP-адрес источника запроса. |
| consumed_from_ip | inet | Необязательно / не указано; IP-адрес источника использования. |
| created_at | timestamptz | Обязательно; Время создания. |

Составные ключи и индексы DBML: `UNIQUE (token_hash)`; `INDEX (phone_id, created_at)`; `INDEX (expires_at) — WHERE consumed_at IS NULL AND revoked_at IS NULL`.

### password_reset_challenges

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| identity_id | uuid | Обязательно; FK; Идентификатор учётной записи Identity. |
| token_hash | text | Обязательно; UNIQUE; Хеш токена. |
| attempts | integer | Обязательно; По умолчанию: 0; Количество попыток. |
| max_attempts | integer | Обязательно; Максимально допустимое число попыток. |
| expires_at | timestamptz | Обязательно; Время истечения. |
| last_attempt_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| consumed_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| revoked_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| requested_from_ip | inet | Необязательно / не указано; IP-адрес источника запроса. |
| consumed_from_ip | inet | Необязательно / не указано; IP-адрес источника использования. |
| created_at | timestamptz | Обязательно; Время создания. |

Составные ключи и индексы DBML: `UNIQUE (token_hash)`; `INDEX (identity_id, created_at)`; `INDEX (expires_at) — WHERE consumed_at IS NULL AND revoked_at IS NULL`.

### oauth_clients

public clients require PKCE. client_id, client_type, auth method and created_at are immutable.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| client_id | text | Обязательно; UNIQUE; Идентификатор OAuth-клиента. |
| client_type | public.oauth_client_type | Обязательно; Тип клиента. |
| token_endpoint_auth_method | public.oauth_token_endpoint_auth_method | Обязательно; Способ аутентификации клиента на token endpoint. |
| name | text | Обязательно; Название или имя. |
| description | text | Необязательно / не указано; Описание. |
| require_pkce | boolean | Обязательно; По умолчанию: true; Признак обязательной проверки PKCE. |
| consent_mode | public.oauth_consent_mode | Обязательно; По умолчанию: remember; Режим согласия. |
| status | public.oauth_client_status | Обязательно; По умолчанию: active; Состояние записи; допустимые переходы описаны в проекте сервиса. |
| created_by_identity_id | uuid | Необязательно / не указано; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| created_at | timestamptz | Обязательно; Время создания. |
| updated_at | timestamptz | Обязательно; Время последнего изменения. |
| disabled_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| deleted_at | timestamptz | Необязательно / не указано; Время удаления. |

Составные ключи и индексы DBML: `UNIQUE (client_id)`; `UNIQUE (id, client_type)`; `INDEX (created_by_identity_id) — WHERE created_by_identity_id IS NOT NULL`.

### oauth_client_redirect_uris

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| oauth_client_id | uuid | Обязательно; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| redirect_uri | text | Обязательно; Разрешённый адрес перенаправления. |
| created_at | timestamptz | Обязательно; Время создания. |

Составные ключи и индексы DBML: `UNIQUE (oauth_client_id, redirect_uri)`.

### oauth_client_post_logout_redirect_uris

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| oauth_client_id | uuid | Обязательно; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| post_logout_redirect_uri | text | Обязательно; Адрес перенаправления после выхода. |
| created_at | timestamptz | Обязательно; Время создания. |

Составные ключи и индексы DBML: `UNIQUE (oauth_client_id, post_logout_redirect_uri)`.

### oauth_client_grant_types

| Поле | Тип | Описание |
|---|---|---|
| oauth_client_id | uuid | Обязательно; PK, FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| grant_type | public.oauth_grant_type | Обязательно; PK; Тип разрешённого OAuth grant. |
| created_at | timestamptz | Обязательно; Время создания. |

Составные ключи и индексы DBML: `PK (oauth_client_id, grant_type)`.

### oauth_client_scopes

Maximum scopes this OAuth client is allowed to request; this is not a user consent.

| Поле | Тип | Описание |
|---|---|---|
| oauth_client_id | uuid | Обязательно; PK, FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| scope | text | Обязательно; PK; Область доступа. |
| created_at | timestamptz | Обязательно; Время создания. |

Составные ключи и индексы DBML: `PK (oauth_client_id, scope)`.

### oauth_client_credentials

Only confidential clients can have credentials. Secret and protocol identity are immutable.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| oauth_client_id | uuid | Обязательно; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| client_type | public.oauth_client_type | Обязательно; FK; По умолчанию: confidential; Тип клиента. |
| secret_hash | text | Обязательно; Хеш секрета. |
| name | text | Необязательно / не указано; Название или имя. |
| created_by_identity_id | uuid | Необязательно / не указано; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| created_at | timestamptz | Обязательно; Время создания. |
| updated_at | timestamptz | Обязательно; Время последнего изменения. |
| expires_at | timestamptz | Необязательно / не указано; Время истечения. |
| revoked_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| revoke_reason | text | Необязательно / не указано; Причина отзыва. |
| last_used_at | timestamptz | Необязательно / не указано; Время последнего использования. |

Составные ключи и индексы DBML: `INDEX (oauth_client_id, created_at)`; `INDEX (created_by_identity_id) — WHERE created_by_identity_id IS NOT NULL`.

### oauth_authorization_grants

Current remembered authorization between one identity and one OAuth client. This is separate from client configuration and from a single authorization transaction.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| identity_id | uuid | Обязательно; FK; Идентификатор учётной записи Identity. |
| oauth_client_id | uuid | Обязательно; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| source | public.oauth_authorization_grant_source | Обязательно; Источник данных. |
| granted_at | timestamptz | Обязательно; Время соответствующего действия или состояния. |
| updated_at | timestamptz | Обязательно; Время последнего изменения. |
| expires_at | timestamptz | Необязательно / не указано; Время истечения. |
| revoked_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |

Составные ключи и индексы DBML: `UNIQUE (identity_id, oauth_client_id)`; `INDEX (identity_id) — WHERE revoked_at IS NULL`; `INDEX (oauth_client_id) — WHERE revoked_at IS NULL`.

### oauth_authorization_grant_scopes

Scopes remembered for the identity-client authorization grant. They must remain a subset of oauth_client_scopes.

| Поле | Тип | Описание |
|---|---|---|
| grant_id | uuid | Обязательно; PK, FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| scope | text | Обязательно; PK; Область доступа. |
| granted_at | timestamptz | Обязательно; Время соответствующего действия или состояния. |

Составные ключи и индексы DBML: `PK (grant_id, scope)`.

### sessions

Authentication identity, client binding and absolute expiration are immutable.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| identity_id | uuid | Обязательно; FK; Идентификатор учётной записи Identity. |
| oauth_client_id | uuid | Обязательно; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| authentication_level | public.authentication_level | Обязательно; Уровень аутентификации. |
| authenticated_methods | text[] | Обязательно; Allowed values: password, totp, webauthn, recovery_code |
| ip_address | inet | Необязательно / не указано; IP-адрес. |
| user_agent | text | Необязательно / не указано; Идентификатор клиентского программного обеспечения. |
| device_name | text | Необязательно / не указано; Название устройства. |
| created_at | timestamptz | Обязательно; Время создания. |
| authenticated_at | timestamptz | Обязательно; Время соответствующего действия или состояния. |
| last_seen_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| idle_expires_at | timestamptz | Обязательно; Время соответствующего действия или состояния. |
| absolute_expires_at | timestamptz | Обязательно; Время соответствующего действия или состояния. |
| revoked_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| revoke_reason | text | Необязательно / не указано; Причина отзыва. |

Составные ключи и индексы DBML: `INDEX (identity_id, created_at)`; `INDEX (oauth_client_id, created_at)`; `INDEX (idle_expires_at) — WHERE revoked_at IS NULL`; `INDEX (absolute_expires_at) — WHERE revoked_at IS NULL`.

### session_scopes

Snapshot of scopes granted when the session was created.

| Поле | Тип | Описание |
|---|---|---|
| session_id | uuid | Обязательно; PK, FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| scope | text | Обязательно; PK; Область доступа. |

Составные ключи и индексы DBML: `PK (session_id, scope)`.

### refresh_token_families

Session binding, creation time and absolute expiration are immutable.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| session_id | uuid | Обязательно; FK, UNIQUE; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| created_at | timestamptz | Обязательно; Время создания. |
| last_rotated_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| absolute_expires_at | timestamptz | Обязательно; Время соответствующего действия или состояния. |
| reuse_detected_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| revoked_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| revoke_reason | text | Необязательно / не указано; Причина отзыва. |

Составные ключи и индексы DBML: `INDEX (absolute_expires_at) — WHERE revoked_at IS NULL`.

### refresh_tokens

Parent must belong to the same family. Token identity and issuance data are immutable.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| family_id | uuid | Обязательно; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| parent_token_id | uuid | Необязательно / не указано; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| token_hash | text | Обязательно; UNIQUE; Хеш токена. |
| issued_at | timestamptz | Обязательно; Время соответствующего действия или состояния. |
| expires_at | timestamptz | Обязательно; Время истечения. |
| rotated_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| revoked_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| revoke_reason | text | Необязательно / не указано; Причина отзыва. |
| reuse_detected_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| created_from_ip | inet | Необязательно / не указано; IP-адрес источника создания. |
| created_user_agent | text | Необязательно / не указано; Клиентское ПО источника создания. |

Составные ключи и индексы DBML: `UNIQUE (id, family_id)`; `UNIQUE (token_hash)`; `UNIQUE (family_id) — WHERE rotated_at IS NULL AND revoked_at IS NULL`; `UNIQUE (parent_token_id) — WHERE parent_token_id IS NOT NULL`; `INDEX (family_id, issued_at)`; `INDEX (expires_at) — WHERE rotated_at IS NULL AND revoked_at IS NULL`.

### signing_keys

public_jwk rejects private JWK parameters; private material is referenced externally.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| kid | text | Обязательно; UNIQUE; Идентификатор ключа подписи. |
| algorithm | text | Обязательно; Идентификатор алгоритма. |
| public_jwk | jsonb | Обязательно; Публичные параметры ключа в формате JWK. |
| private_key_ref | text | Обязательно; Ссылка на приватный ключ. |
| status | public.signing_key_status | Обязательно; По умолчанию: pending; Состояние записи; допустимые переходы описаны в проекте сервиса. |
| created_at | timestamptz | Обязательно; Время создания. |
| not_before | timestamptz | Обязательно; Самое раннее допустимое время. |
| activated_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| retiring_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| retired_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| expires_at | timestamptz | Необязательно / не указано; Время истечения. |
| compromised_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |

Составные ключи и индексы DBML: `UNIQUE (kid)`; `UNIQUE (algorithm) — WHERE status = 'active'`; `INDEX (status, not_before, expires_at) — WHERE status IN ('active', 'retiring')`.

### service_accounts

organization_id, account_key and created_at are immutable.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| organization_id | uuid | Обязательно; Owned by Organization Service |
| account_key | text | Обязательно; Ключ сервисной учётной записи. |
| name | text | Обязательно; Название или имя. |
| description | text | Необязательно / не указано; Описание. |
| status | public.service_account_status | Обязательно; По умолчанию: active; Состояние записи; допустимые переходы описаны в проекте сервиса. |
| created_by_identity_id | uuid | Необязательно / не указано; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| created_at | timestamptz | Обязательно; Время создания. |
| updated_at | timestamptz | Обязательно; Время последнего изменения. |
| disabled_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| deleted_at | timestamptz | Необязательно / не указано; Время удаления. |

Составные ключи и индексы DBML: `UNIQUE (organization_id, account_key)`; `INDEX (organization_id, id) — WHERE status = 'active'`; `INDEX (deleted_at) — WHERE deleted_at IS NOT NULL`; `INDEX (created_by_identity_id) — WHERE created_by_identity_id IS NOT NULL`.

### service_account_credentials

Credential binding, type, secret hash and created_at are immutable.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| service_account_id | uuid | Обязательно; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| credential_type | public.service_account_credential_type | Обязательно; По умолчанию: shared_secret; Тип учётных данных. |
| secret_hash | text | Обязательно; Хеш секрета. |
| secret_prefix | text | Необязательно / не указано; Префикс секрета для идентификации. |
| name | text | Необязательно / не указано; Название или имя. |
| created_by_identity_id | uuid | Необязательно / не указано; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| created_at | timestamptz | Обязательно; Время создания. |
| updated_at | timestamptz | Обязательно; Время последнего изменения. |
| expires_at | timestamptz | Необязательно / не указано; Время истечения. |
| revoked_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| revoke_reason | text | Необязательно / не указано; Причина отзыва. |
| last_used_at | timestamptz | Необязательно / не указано; Время последнего использования. |

Составные ключи и индексы DBML: `INDEX (service_account_id, created_at)`; `INDEX (created_by_identity_id) — WHERE created_by_identity_id IS NOT NULL`; `INDEX (secret_prefix) — WHERE secret_prefix IS NOT NULL AND revoked_at IS NULL`.

### security_events

Actor type determines whether identity_id or service_account_id must be present.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| event_type | text | Обязательно; Тип события. |
| result | public.security_event_result | Обязательно; Результат операции. |
| actor_type | public.security_actor_type | Обязательно; Тип инициатора. |
| identity_id | uuid | Необязательно / не указано; FK; Идентификатор учётной записи Identity. |
| service_account_id | uuid | Необязательно / не указано; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| session_id | uuid | Необязательно / не указано; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| oauth_client_id | uuid | Необязательно / не указано; FK; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| identifier_hash | bytea | Необязательно / не указано; Хеш идентификатора. |
| ip_address | inet | Необязательно / не указано; IP-адрес. |
| user_agent | text | Необязательно / не указано; Идентификатор клиентского программного обеспечения. |
| correlation_id | uuid | Необязательно / не указано; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| occurred_at | timestamptz | Обязательно; Время события. |
| metadata | jsonb | Обязательно; По умолчанию: {}; Дополнительные технические метаданные. |

Составные ключи и индексы DBML: `INDEX (identity_id, occurred_at) — WHERE identity_id IS NOT NULL`; `INDEX (service_account_id, occurred_at) — WHERE service_account_id IS NOT NULL`; `INDEX (session_id, occurred_at) — WHERE session_id IS NOT NULL`; `INDEX (oauth_client_id, occurred_at) — WHERE oauth_client_id IS NOT NULL`; `INDEX (event_type, occurred_at)`; `INDEX (identifier_hash, occurred_at) — WHERE identifier_hash IS NOT NULL`; `INDEX (occurred_at)`; `INDEX (correlation_id, occurred_at) — WHERE correlation_id IS NOT NULL`.

### outbox_messages

Transactional outbox. Payload and headers are JSON objects; processing state controls retries and publication.

| Поле | Тип | Описание |
|---|---|---|
| id | uuid | Обязательно; PK; Идентификатор записи. |
| event_id | uuid | Обязательно; UNIQUE; Идентификатор события. |
| aggregate_type | varchar(255) | Обязательно; Тип агрегата — источника события. |
| aggregate_id | uuid | Обязательно; Идентификатор связанного объекта; владелец и назначение определяются контекстом сервиса. |
| aggregate_version | bigint | Необязательно / не указано; Версия агрегата — источника события. |
| event_type | varchar(255) | Обязательно; Тип события. |
| payload | jsonb | Обязательно; Содержимое события или операции. |
| headers | jsonb | Обязательно; По умолчанию: '{}'::jsonb; Заголовки события. |
| partition_key | varchar(255) | Необязательно / не указано; Ключ распределения события. |
| status | varchar(32) | Обязательно; По умолчанию: PENDING; Состояние записи; допустимые переходы описаны в проекте сервиса. |
| attempts | integer | Обязательно; По умолчанию: 0; Количество попыток. |
| occurred_at | timestamptz | Обязательно; Время события. |
| processing_started_at | timestamptz | Необязательно / не указано; Время соответствующего действия или состояния. |
| published_at | timestamptz | Необязательно / не указано; Время публикации. |
| next_attempt_at | timestamptz | Необязательно / не указано; Время следующей попытки. |
| last_error | text | Необязательно / не указано; Последняя зафиксированная ошибка. |
| created_at | timestamptz | Обязательно; По умолчанию: now(); Время создания. |
| updated_at | timestamptz | Обязательно; По умолчанию: now(); Время последнего изменения. |

Составные ключи и индексы DBML: `UNIQUE (event_id)`; `INDEX (aggregate_type, aggregate_id, aggregate_version)`; `INDEX (status, next_attempt_at, occurred_at) — WHERE status IN ('PENDING', 'FAILED')`; `INDEX (processing_started_at) — WHERE status = 'PROCESSING'`; `INDEX (partition_key, occurred_at) — WHERE status = 'PENDING'`; `INDEX (published_at) — WHERE status = 'PUBLISHED'`.

## Перечисления

Допустимые значения перечислимых типов из DBML. Переходы между состояниями задаются правилами сервиса.

| Тип | Допустимые значения |
|---|---|
| identity_status | pending, active, disabled, deleted |
| identity_profile_source | self_declared, provisioned, verified |
| identity_assurance_level | self_declared, organization_asserted, identity_verified |
| mfa_factor_type | totp, webauthn |
| totp_algorithm | sha1, sha256, sha512 |
| authentication_level | single_factor, multi_factor |
| oauth_client_type | public, confidential |
| oauth_consent_mode | required, remember, trusted |
| oauth_authorization_grant_source | user_consent, trusted_client_policy |
| oauth_token_endpoint_auth_method | none, client_secret_basic |
| oauth_grant_type | authorization_code, refresh_token |
| oauth_client_status | active, disabled, deleted |
| signing_key_status | pending, active, retiring, retired, compromised |
| service_account_status | active, disabled, deleted |
| service_account_credential_type | shared_secret |
| security_event_result | success, failure, blocked |
| security_actor_type | anonymous, identity, service_account, administrator, system |

