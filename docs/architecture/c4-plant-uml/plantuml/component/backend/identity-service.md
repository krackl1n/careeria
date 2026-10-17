# Identity Service — компоненты L3

Сервис владеет учётными записями пользователей платформы Careeria, аутентификацией, хранением учётных данных, выпуском и верификацией криптографических токенов, управлением пользовательскими сессиями и ротацией refresh-токенов, а также интеграцией с внешним Identity Provider. Сервис не хранит бизнес-данные резюме кандидата, компаний, вакансий или процессов найма. Доступ к платформенным ресурсам авторизуется на основе выданных токенов и контекста идентичности.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/component/backend/identity-service.puml
@enduml
```

## Статус описания и границы ответственности

Внутри контейнера выделены ключевые прикладные и инфраструктурные модули:

- **Authentication & Registration Service** — координация сценариев регистрации identity, проверки паролей, прохождения верификационных челленджей, двухфакторной аутентификации и предоставления согласий .
- **OAuth & Token Service** — реализация протоколов OAuth 2.0 и OpenID Connect Core 1.0: инициализация авторизации, обмен одноразовых authorization codes на токены, ротация refresh-токенов с обнаружением повторного использования, отзыв токенов и выдача UserInfo.
- **External Integration Adapter** — адаптер для внешних интеграций: реализация OIDC-протокола для федеративного единого входа через внешний Identity Provider, проверки внешних ID-токенов по публичному JWKS и связывания внешних субъектов с локальными identity Careeria.
- **Token Minting & Key Service** — криптографическая подпись Access Tokens и ID Tokens алгоритмом Ed25519, управление жизненным циклом приватных ключей и публикация набора публичных ключей JWKS.
- **Credential Security Service** — безопасное хеширование и верификация паролей алгоритмом Argon2id, защита от подбора, криптографическая генерация случайных токенов высокой энтропии и их хеширование SHA-256 для безопасного хранения.

Это компоненты одного контейнера, а не отдельно развёртываемые микросервисы. Все они работают in-process в одном рантайме Go.

| Данные / действие | Владелец | Граница ответственности |
|---|---|---|
| Учётные записи, email, телефоны, профили identity, учётные данные | Identity Service | Не хранит резюме, опыт работы, навыки, отклики или реквизиты юридических лиц. |
| Сессии, refresh token families, история ротаций, токены доступа | Identity Service | Управляет жизненным циклом аутентификационных сессий; не принимает решений о ReBAC-правах внутри компаний. |
| Федеративная аутентификация, корпоративный SSO, внешний каталог пользователей | Identity Provider | Внешний OIDC IdP; Identity Service проверяет OIDC assertions и связывает external subject. |
| Профиль кандидата, навыки, опыт, портфолио, видимость для рекрутеров | Candidate Service | Привязывается к `identity.user_id`, но управляется независимо в Candidate DB. |
| Компании, членство сотрудников, корпоративные роли | Company Service | Сведения об участии в компаниях хранятся в Company DB; роли синхронизируются в OpenFGA. |
| Отношения доступа, проверка прав на объекты найма | OpenFGA / AuthZ Projection | Права доступа проверяются через OpenFGA на основе доверенного identity context из проверенного токена. |
| Доставка кодов подтверждения | Notification Service | Identity Service генерирует код челленджа и фиксирует событие в outbox; прямую отправку писем и SMS выполняет Notification Service. |

Внутренние компоненты и их связи объявлены в `identity-service.puml`. Контейнер Identity Service, Identity DB и Identity Redis определены в `model/identity-elements.puml` для L2/L3. Общие контейнеры подключаются из `model/component-elements.puml`. Все внутренние инфраструктурные элементы находятся внутри границы системы Careeria и окрашены серым цветом готовой платформенной инфраструктуры, а внешний Identity Provider вынесен за границу системы Careeria. Kafka, Schema Registry и Debezium участвуют в CDC-маршруте платформы на L2, но не являются прямыми клиентами рантайма Identity Service: сервис фиксирует события в transactional outbox, а надёжную выгрузку выполняет Debezium CDC через чтение WAL PostgreSQL.

Репозитории и outbox получают общий транзакционный контекст в единой SQL-транзакции PostgreSQL; отдельного компонента Tx Manager на диаграмме нет — это техническая обвязка `WithinTransaction`.

## Элементы диаграммы

### Компоненты внутри Identity Service

| Элемент | Технология / слой | Ответственность | Основные зависимости |
|---|---|---|---|
| gRPC Server | Go / gRPC, входной адаптер | Предоставляет API сервисов `RegistrationService`, `AuthenticationService`, `OAuthService`, `IdentitySessionService`, `IdentityMetadataService` и Admin API. Проверяет структуру входных DTO, валидирует форматы, переводит доменные ошибки в gRPC Status Codes. | Authentication & Registration Service, OAuth & Token Service |
| Authentication & Registration Service | Go Application Service | Координирует сценарии регистрации новых пользователей, аутентификации по логину и паролю, верификации контактов (email/телефон), прохождения шагов MFA и фиксации consent. | Credential Security Service, External Integration Adapter, Identity Repo, Credential & Auth State Repo, Authorization Transaction Repo, Transactional Outbox |
| OAuth & Token Service | Go Application Service | Координирует шаги OAuth 2.0 / OIDC авторизации: `Authorize`, `ContinueAuthorization`, обмен authorization code на токены, обмен refresh-токена с ротацией семейства, обнаружение reuse, отзыв токенов и сборку UserInfo. | Token Minting & Key Service, Credential Security Service, Session & Token Family Repo, OAuth Client & Keys Repo, Authorization Transaction Repo, Transactional Outbox |
| External Integration Adapter | Go / HTTP, OIDC, Adapter | Адаптер для внешних интеграций: взаимодействие с внешним Identity Provider по OIDC, валидация внешних токенов, запрос UserInfo, связывание external subjects с локальными identities платформы. | Identity Provider |
| Token Minting & Key Service | Go / JWT, Ed25519 | Формирует полезную нагрузку JWT Access Token по профилю RFC 9068 и OIDC ID Token, подписывает приватным Ed25519 ключом, публикует JWKS и осуществляет ротацию ключей. | OAuth Client & Keys Repo |
| Credential Security Service | Go / Argon2id, Crypto, SHA-256 | Реализует хеширование паролей Argon2id с криптографической солью, проверку хешей за константное время, генерацию случайных криптографических токенов и хеширование SHA-256 для refresh-токенов и authorization codes. | *Автономный криптографический сервис* |
| Identity Repo | Go / pgx, sqlc, Repository | Сохраняет и загружает агрегаты `Identity`, подтверждённые и основные `Email` и `Phone`, а также связи внешних провайдеров `ExternalIdentityAccount`. | Identity DB |
| Credential & Auth State Repo | Go / pgx, sqlc, Repository | Сохраняет хеши паролей `IdentityPasswordCredential`, отслеживает счётчик неудачных попыток входа и временную блокировку аккаунта в `IdentityPasswordAuthState`. | Identity DB |
| Session & Token Family Repo | Go / pgx, sqlc, Repository | Сохраняет сущности `Session`, назначенные `SessionScopes`, агрегаты `RefreshTokenFamily` и цепочки одноразовых `RefreshToken`. Выполняет атомарную блокировку семейства токенов при ротации. | Identity DB |
| OAuth Client & Keys Repo | Go / pgx, sqlc, Repository | Загружает настройки зарегистрированных OAuth-клиентов, разрешённые redirect URIs, scopes, ранее выданные пользователем гранты и активные/устаревающие signing keys. | Identity DB |
| Authorization Transaction Repo | Go / Redis, Repository | Хранит в Redis краткоживущие транзакции авторизации `AuthorizationTransaction` и одноразовые authorization codes `AuthorizationCode` с атомарным погашением. | Identity Redis |
| Transactional Outbox | Go / pgx, sqlc | Фиксирует интеграционные и аудиторные события безопасности в PostgreSQL в той же транзакции, что и изменение доменного состояния. Доставка в Kafka выполняется платформенным Debezium CDC через чтение PostgreSQL WAL. | Identity DB |

### Соседние контейнеры и границы

| Элемент | Принадлежность | Роль в сценариях |
|---|---|---|
| Careeria | Граница нашей системы | Включает прикладные контейнеры и платформенную инфраструктуру системы, отделяя их от внешних систем. |
| Identity Service | Прикладной контейнер Careeria | Владеет учётными записями, credentials, сессиями и токенами; компоненты внутри его границы вызывают друг друга in-process. |
| API Gateway | Наш прикладной контейнер | Проверяет JWT Access Tokens локально по JWKS или проксирует вызовы авторизации в Identity Service по gRPC. |
| Identity DB | Наша инфраструктура, PostgreSQL | Реляционная СУБД для постоянного хранения пользователей, хешей паролей, профилей, сессий, истории ротаций refresh-токенов, клиентов, ключей и outbox-событий. |
| Identity Redis | Наша инфраструктура, Redis | In-memory хранилище с TTL для промежуточных транзакций авторизации и одноразовых authorization codes. |
| Identity Provider | Внешняя система вне границы Careeria | Внешний OIDC IdP: обеспечивает корпоративную федерацию, единую точку входа и внешнюю аутентификацию пользователей. Расположен вне границы системы Careeria. |

## Каталог юзкейсов

| № | Юзкейс | Прикладной компонент | Статус контракта | Итоговый результат |
|---|---|---|---|---|
| 1 | `RegisterIdentity` | Authentication & Registration Service | Protobuf `RegistrationService.Register` | Регистрация нового identity по email/телефону с паролем и профилем в рамках OAuth-транзакции. |
| 2 | `GetAuthenticationMethods` | Authentication & Registration Service | Protobuf `AuthenticationService.GetAuthenticationMethods` | Возврат списка включённых способов входа. |
| 3 | `GetAuthenticationContext` | Authentication & Registration Service | Protobuf `AuthenticationService.GetAuthenticationContext` | Получение текущего состояния транзакции авторизации и запрошенных OAuth-клиентом scopes. |
| 4 | `AuthenticateWithPassword` | Authentication & Registration Service | Protobuf `AuthenticationService.AuthenticateWithPassword` | Проверка пароля через Argon2id с учётом попыток и блокировки; привязка identity к транзакции. |
| 5 | `VerifyEmail` | Authentication & Registration Service | Protobuf `AuthenticationService.VerifyEmail` | Валидация 6-значного OTP-кода подтверждения email; перевод контакта и identity в активный статус. |
| 6 | `VerifyPhone` | Authentication & Registration Service | Protobuf `AuthenticationService.VerifyPhone` | Валидация 6-значного OTP-кода подтверждения телефона; активация контакта и identity. |
| 7 | `SkipMFAEnrollment` | Authentication & Registration Service | Protobuf `AuthenticationService.SkipMFAEnrollment` | Пропуск настройки второго фактора для политик с допустимым `single_factor`. |
| 8 | `GrantAuthorizationConsent` | Authentication & Registration Service | Protobuf `AuthenticationService.GrantConsent` | Подтверждение пользователем прав доступа для OAuth-клиента; сохранение постоянного согласия. |
| 9 | `DenyAuthorizationConsent` | Authentication & Registration Service | Protobuf `AuthenticationService.DenyConsent` | Отклонение запроса на выдачу прав; прерывание процесса с ошибкой `access_denied`. |
| 10 | `Authorize` | OAuth & Token Service | RFC 6749 / OIDC `AuthorizeEndpoint` | Инициализация OAuth 2.0 Authorization Code Flow с PKCE: валидация клиента, redirect URI и создание транзакции в Redis. |
| 11 | `ContinueAuthorization` | OAuth & Token Service | RFC 6749 / OIDC `ContinueEndpoint` | Завершение интерактивного входа: выпуск одноразового `authorization_code` и возврат redirect URL. |
| 12 | `ExchangeToken (code)` | OAuth & Token Service | RFC 6749 / RFC 7636 PKCE `TokenEndpoint` | Обмен `authorization_code` с проверкой PKCE `code_verifier`: создание Session и выпуск Access + ID + Refresh токенов. |
| 13 | `ExchangeToken (refresh)` | OAuth & Token Service | RFC 6749 / Token Family Rotation `TokenEndpoint` | Ротация сессии: проверка токена обновления, обнаружение reuse, выдача нового refresh-токена и Access Token. |
| 14 | `RevokeToken` | OAuth & Token Service | RFC 7009 `RevocationEndpoint` | Отзыв refresh-токена или сессии по стандарту RFC 7009; идемпотентное подтверждение. |
| 15 | `GetUserInfo` | OAuth & Token Service | OpenID Connect Core 1.0 `UserInfoEndpoint` | OIDC UserInfo endpoint: валидация Bearer Access Token и отдача разрешённых claims пользователя. |
| 16 | `GetIdentitySession` | OAuth & Token Service | Protobuf `IdentitySessionService.GetSession` | Получение метаданных текущей активной сессии. |
| 17 | `LogoutIdentitySession` | OAuth & Token Service | Protobuf `IdentitySessionService.LogoutSession` | Явный выход пользователя: отзыв сессии и связанной refresh token family с фиксацией в audit log. |
| 18 | `GetOpenIDConfiguration` | Token Minting & Key Service | RFC 8414 / OIDC Discovery | OpenID Connect Discovery: отдача конфигурации провайдера (`/.well-known/openid-configuration`). |
| 19 | `GetJSONWebKeySet` | Token Minting & Key Service | RFC 7517 JWKS | Публикация открытых ключей подписи JWKS (`/.well-known/jwks.json`) для валидации токенов без сетевых обращений. |
| 20 | `FederatedOidcLogin` | External Integration Adapter | OIDC Federation Flow | Федеративный вход через внешний Identity Provider: проверка внешнего OIDC ID-токена, связывание аккаунта и выпуск Careeria-сессии. |

## Подробные сценарии

### RegisterIdentity — регистрация новой учётной записи

- **Вход и инициатор:** неаутентифицированный пользователь через Web UI. Параметры запроса: идентификатор транзакции авторизации (`authorization_transaction_id`), один контакт (email или телефон в формате E.164), открытый пароль (8–1024 байт), базовый профиль (фамилия, имя, отчество), IP-адрес и User-Agent.
- **Ход сценария:**
  1. gRPC Server валидирует входной DTO и передаёт команду в Authentication & Registration Service.
  2. Сервис проверяет формат контакта (lowercase и trim для email, регулярное выражение `^\\+[1-9][0-9]{7,14}$` для телефона) и минимальную сложность пароля.
  3. `CredentialSecurityService` генерирует 16-байтную криптографическую соль (`crypto/rand`) и вычисляет хеш пароля алгоритмом Argon2id (m=65536, t=3, p=4).
  4. Сервис загружает и валидирует `AuthorizationTransaction` из Redis. Затем в одной SQL-транзакции PostgreSQL:
     - Создаётся запись `Identity` (`status = 'pending'`);
     - Сохраняется контакт `IdentityEmail` или `IdentityPhone` (`is_primary = true`);
     - Сохраняется `IdentityProfile` с версией 1;
     - Сохраняется `IdentityPasswordCredential` с хешем Argon2id;
     - Сохраняется уникальный `authorization_transaction_id` для идемпотентного повтора регистрации;
     - Генерируется 6-значный криптографический код верификации (срок действия 10 минут) и фиксируется в таблице челленджей;
     - В таблицу `outbox` записывается интеграционное событие `IdentityCreated`.
  5. После SQL-коммита сервис обновляет транзакцию в Redis: устанавливает `identity_id` и `next_step = EMAIL_VERIFICATION` (или `PHONE_VERIFICATION`). Если обновление Redis не удалось, повтор запроса с тем же `authorization_transaction_id` находит уже созданную Identity и завершает обновление шага без повторной записи в БД.
- **Результат:** новая учётная запись сохранена в статусе `pending`, сгенерирован OTP-челлендж для подтверждения контакта, клиенту возвращён следующий шаг процесса регистрации.
- **Ошибки и ограничения:** при совпадении с уже подтверждённым контактом возвращается `AlreadyExists` (`registration_contact_already_exists`). Брошенная регистрация не засоряет активные учётные записи платформы и автоматически истекает по TTL в Redis.

### AuthenticateWithPassword — аутентификация по логину и паролю

- **Вход и инициатор:** пользователь, проходящий авторизацию. Вход: `authorization_transaction_id`, логин (email или телефон), открытый пароль, IP-адрес и User-Agent.
- **Ход сценария:**
  1. Сервис загружает `Identity` по контакту и проверяет, что статус равен `active` (статусы `disabled` или `deleted` возвращают ошибку `account_disabled` / `identity_not_found`).
  2. Загружается `IdentityPasswordAuthState`. Если `locked_until > now`, сценарий немедленно прерывается с ошибкой `PermissionDenied` (`account_locked`) без вычисления хеша пароля.
  3. Загружается `IdentityPasswordCredential`. `CredentialSecurityService` проверяет пароль против сохранённого хеша Argon2id за константное время (`subtle.ConstantTimeCompare`).
  4. При несовпадении: увеличивается счётчик `failed_attempts`. При достижении 5 неудачных попыток выставляется блокировка на 15 минут (`locked_until = now + 15m`). Факт фиксируется в таблице `security_events` со статусом `FAILURE`. Возвращается ошибка `Unauthenticated` (`invalid_credentials`).
  5. При совпадении: счётчик неудач сбрасывается в 0, очищается блокировка, фиксируется событие `SUCCESS`. В Redis обновляется `AuthorizationTransaction`: устанавливается `authenticated_identity_id` и способ `password`.
  6. Вычисляется следующий шаг: при необходимости второго фактора — `NEXT_STEP_MFA`, при запросе новых scopes — `NEXT_STEP_CONSENT`, иначе — `NEXT_STEP_CONTINUE`.
- **Результат:** пароль успешно проверен, транзакция переведена к следующему шагу интерактивного входа. Пароль немедленно удаляется из памяти процесса.
- **Ошибки и ограничения:** `Unauthenticated` при ошибке учётных данных, `PermissionDenied` при временной блокировке.

### VerifyEmail и VerifyPhone — подтверждение первичных контактов

- **Вход и инициатор:** пользователь; вход: `authorization_transaction_id`, 6-значный числовой OTP-код.
- **Ход сценария:**
  1. Сервис загружает активный челлендж, сверяет переданный код с сохранённым криптографическим хешем, проверяет срок действия (10 минут) и счётчик попыток (не более 5).
  2. В единой SQL-транзакции PostgreSQL:
     - Контакт помечается как `verified_at = now`;
     - Статус `Identity` переводится из `pending` в `active`;
     - В таблицу `outbox` записывается событие `IdentityVerified`.
  3. После SQL-коммита сервис обновляет транзакцию авторизации в Redis: шаг переходит к выдаче consent либо завершению входа (`next_step = CONTINUE`). При сбое Redis повторная проверка статуса Identity восстанавливает этот шаг без повторного события `IdentityVerified`.
- **Результат:** контакт и учётная запись активированы, пользователь может переходить к выпуску токенов.
- **Ошибки и ограничения:** `InvalidArgument` при неверном или просроченном коде. Доставка SMS/Email выполняется Notification Service через чтение outbox коннектором Debezium CDC.

### GetAuthenticationMethods и GetAuthenticationContext — метаданные процесса входа

- **Вход и инициатор:** Web UI / BFF; вход: `authorization_transaction_id`.
- **Ход сценария:**
  1. `GetAuthenticationMethods` возвращает перечень разрешённых способов аутентификации (локальный пароль, внешний Keycloak SSO, MFA).
  2. `GetAuthenticationContext` считывает текущее состояние транзакции из Redis через `AuthorizationTransactionRepo` и возвращает клиенту метаданные: текущий шаг (`next_step`), запрошенные OAuth-scopes, параметры клиента и статус подтверждения identity.
- **Результат:** фронтенд Careeria получает данные для рендеринга соответствующего экрана авторизации (ввод логина, подтверждение email, экран согласия).
- **Ошибки и ограничения:** `NotFound`, если срок жизни транзакции в Redis (10 минут) истёк; в этом случае пользователь направляется на перезапуск OAuth Flow.

### SkipMFAEnrollment — пропуск настройки второго фактора

- **Вход и инициатор:** пользователь на экране предложения настройки второго фактора. Вход: `authorization_transaction_id`.
- **Ход сценария:** сервис проверяет политику OAuth-клиента и требуемый уровень аутентификации. Если политика допускает `single_factor`, транзакция в Redis переводится на следующий шаг (`next_step = CONSENT` или `CONTINUE`).
- **Результат:** процесс авторизации продолжается с `authentication_level = single_factor`.
- **Ошибки и ограничения:** `PermissionDenied`, если для клиента или запрашиваемого контекста задано жесткое требование многофакторной аутентификации (`acr = multi_factor`).

### GrantAuthorizationConsent и DenyAuthorizationConsent — управление согласиями

- **Вход и инициатор:** аутентифицированный пользователь на экране подтверждения прав клиентского приложения. Вход: `authorization_transaction_id`, перечень подтверждённых scopes (или отказ).
- **Ход сценария:**
  1. При `GrantAuthorizationConsent`: сервис сохраняет постоянный пользовательский грант в PostgreSQL (`oauth_client_user_grants`), а транзакцию в Redis переводит в состояние `next_step = CONTINUE`.
  2. При `DenyAuthorizationConsent`: транзакция в Redis помечается как отклонённая (`access_denied`).
- **Результат:** при согласии клиент готов получить authorization code; при отказе процесс прерывается с возвратом на redirect URI клиента с ошибкой `access_denied` в соответствии с RFC 6749.
- **Ошибки и ограничения:** отказ пользователя является стандартным исходом сценария OAuth 2.0.

### Authorize — инициализация OAuth 2.0 / OIDC Authorization Flow

- **Вход и инициатор:** клиентское приложение (Careeria BFF или внешний клиент). Параметры HTTP/gRPC запроса: `client_id`, `redirect_uri`, `response_type = code`, `scope`, `state`, `code_challenge`, `code_challenge_method = S256`.
- **Ход сценария:**
  1. `OAuthClientRepo` загружает клиента из PostgreSQL, проверяет статус `active`, соответствие `redirect_uri` белому списку и допустимость запрошенных scopes.
  2. Проверяется наличие PKCE: для публичных клиентов параметр `code_challenge` с методом `S256` является строго обязательным.
  3. В Redis создаётся запись `AuthorizationTransaction` с уникальным UUID и TTL 10 минут.
- **Результат:** сформирован URL перенаправления на UI аутентификации Careeria (`/login?transaction_id=...`).
- **Ошибки и ограничения:** `invalid_client` при невалидном клиенте, `invalid_request` при отсутствии PKCE или несоответствии redirect URI.

### ContinueAuthorization — завершение интерактивного входа и выдача кода

- **Вход и инициатор:** пользователь после успешного прохождения всех интерактивных этапов UI (пароль, верификация, consent). Вход: `authorization_transaction_id`.
- **Ход сценария:**
  1. Сервис проверяет в Redis готовность транзакции к завершению (`next_step = CONTINUE`, подтверждённый `identity_id`).
  2. `CredentialSecurityService` генерирует криптографически стойкий случайный `authorization_code` (256 бит энтропии).
  3. Код сохраняется в Redis (`AuthorizationCode`) с жестким TTL 2 минуты, связывая `identity_id`, `client_id`, `redirect_uri`, `scopes`, `code_challenge`.
  4. Транзакция авторизации удаляется из Redis.
- **Результат:** возврат redirect URI клиенту с параметрами `code` и исходным `state`.
- **Ошибки и ограничения:** `invalid_step` при незавершённых шагах; `transaction_expired` при превышении времени сессии.

### ExchangeToken (authorization_code) — обмен кода на токены с PKCE

- **Вход и инициатор:** backend OAuth-клиента (например, Careeria BFF). Параметры: `grant_type = authorization_code`, `code`, `redirect_uri`, `client_id`, `client_secret` (для confidential clients), `code_verifier` (PKCE).
- **Ход сценария:**
  1. Аутентификация клиента: проверка секрета либо подтверждение типа `public`.
  2. Атомарное извлечение и удаление кода из Redis командой `GETDEL auth_code:{code}`. Если код не найден — немедленная ошибка `invalid_grant`.
  3. Проверка PKCE: вычисляется `BASE64URL(SHA256(code_verifier))` и сверяется за константное время с сохранённым `code_challenge`. При несовпадении — отказ `invalid_grant`.
  4. В единой SQL-транзакции PostgreSQL:
     - Создаётся запись `Session` (`idle_expires_at = now + 24h`, `absolute_expires_at = now + 720h`);
     - Создаётся `RefreshTokenFamily` для данной сессии;
     - `CredentialSecurityService` генерирует raw refresh-токен высокой энтропии и вычисляет его SHA-256 хеш;
     - Сохраняется первый `RefreshToken` (`parent_token_id = NULL`);
     - В таблицу `outbox` записывается событие `SessionCreated`.
  5. `TokenMintingService` подписывает приватным Ed25519 ключом:
     - **Access Token:** JWT RFC 9068 (`typ: at+jwt`), алгоритм `EdDSA`, срок действия 15 минут;
     - **ID Token:** JWT OIDC (`typ: JWT`), содержащий стандартные claims пользователя.
- **Результат:** ответ клиенту с `access_token`, `refresh_token`, `id_token`, `token_type = Bearer`, `expires_in = 900`.
- **Ошибки и ограничения:** `invalid_grant` при неверном/повторно предъявленном коде или ошибке PKCE. Однократное использование кода гарантируется атомарной операцией `GETDEL`.

### ExchangeToken (refresh_token) — ротация сессии и обнаружение reuse

- **Вход и инициатор:** клиентское приложение, желающее продлить доступ. Вход: `grant_type = refresh_token`, `refresh_token`, `client_id`, `client_secret`.
- **Ход сценария:**
  1. Сервис вычисляет `SHA-256(refresh_token)`.
  2. В транзакции PostgreSQL выполняется выборка с блокировкой строки: `RefreshTokenFamily` и запись предъявленного токена блокируются через `FOR UPDATE`.
  3. **Обнаружение повторного использования (Reuse Detection):**
     - Если найденный токен имеет `rotated_at IS NOT NULL` или `revoked_at IS NOT NULL`:
       - Зафиксирована атака повторного использования украденного токена!
       - Семейство токенов и связанная сессия немедленно отзываются (`revoked_at = now`, `reason = 'refresh_token_reuse_detected'`);
       - В `outbox` записываются критические события `RefreshTokenReuseDetected` и `SessionRevoked`;
       - В таблицу `security_events` заносится запись инцидента безопасности;
       - Транзакция коммитится, клиенту возвращается ошибка `invalid_grant`.
  4. **Успешная ротация (токен активен):**
     - Проверяется активность сессии (`now < session.idle_expires_at` и `now < session.absolute_expires_at`);
     - Предъявленный токен помечается как погашенный: `rotated_at = now`;
     - Генерируется новый raw refresh-токен, вычисляется его SHA-256 хеш;
     - Создаётся новая запись `RefreshToken` с `parent_token_id = presented_token_id`;
     - В сессии сдвигается `idle_expires_at = min(now + 24h, session.absolute_expires_at)`;
     - В `outbox` фиксируется событие `TokenRotated`;
     - `TokenMintingService` выпускает свежий JWT Access Token на 15 минут.
- **Результат:** клиенту возвращаются свежий `access_token` и **новый** `refresh_token`. Использованный refresh-токен погашен навсегда.
- **Ошибки и ограничения:** `invalid_grant` при reuse, истечении сессии или блокировке.

### RevokeToken — отзыв токена и сессии (RFC 7009)

- **Вход и инициатор:** клиентское приложение при выходе; вход: `token`, необязательный `token_type_hint`.
- **Ход сценария:** сервис хеширует токен SHA-256, находит соответствующий `RefreshToken` и связанную `Session`. Помечает токен и сессию как отозванные (`revoked_at = now`, `revoke_reason = 'client_revocation'`). Событие `SessionRevoked` пишется в outbox.
- **Результат:** по стандарту RFC 7009, если токен найден и отозван или уже был отозван ранее, возвращается успешный HTTP 200 / gRPC OK (не раскрывая факт существования токена третьим лицам).
- **Ошибки и ограничения:** операция идемпотентна.

### GetUserInfo — выдача OIDC профиля пользователя

- **Вход и инициатор:** клиент с Bearer Access Token в заголовке `Authorization`.
- **Ход сценария:** токен проверяется локально по открытому Ed25519 ключу (подпись, issuer, audience, срок действия). Из claims извлекаются `sub` (identity ID) и scopes. Сервис загружает `IdentityProfile` и подтверждённые контакты. Формирует JSON с полями `sub`, `email`, `email_verified`, `phone_number`, `name`, `family_name`, `given_name` в строгом соответствии с запрошенными scopes.
- **Результат:** стандартный JSON профиля пользователя по спецификации OpenID Connect Core 1.0.
- **Ошибки и ограничения:** `Unauthenticated` при истёкшем или невалидном токене доступа.

### GetIdentitySession и LogoutIdentitySession — управление активной сессией

- **Вход и инициатор:** аутентифицированный пользователь или BFF. Вход: `session_id`.
- **Ход сценария:**
  1. `GetIdentitySession` загружает метаданные сессии из `SessionRepo`: IP-адрес, User-Agent, уровень входа, методы, сроки действия (`idle_expires_at`, `absolute_expires_at`).
  2. `LogoutIdentitySession` переводит сессию и связанное семейство refresh-токенов в статус `revoked` (`revoke_reason = 'user_logout'`). В `outbox` фиксируется событие `SessionRevoked`, в аудит — запись выхода.
- **Результат:** при logout сессия немедленно прекращается; все последующие попытки обновить токены этой сессии отклоняются.
- **Ошибки и ограничения:** `NotFound` при отсутствии сессии; повторный logout идемпотентен.

### GetOpenIDConfiguration и GetJSONWebKeySet — публикация OIDC Discovery и JWKS

- **Вход и инициатор:** API Gateway, BFF, внешние клиенты.
- **Ход сценария:**
  1. `GetOpenIDConfiguration` отдаёт стандартный JSON провайдера (`/.well-known/openid-configuration`): issuer, token endpoint, jwks_uri, поддерживаемые scopes и алгоритмы (`EdDSA`).
  2. `GetJSONWebKeySet` загружает активные и ротируемые ключи из `SigningKey` и отдаёт JWKS (`/.well-known/jwks.json`), содержащий только открытые компоненты ключей (Ed25519 public key, `crv: Ed25519`).
- **Результат:** API Gateway и клиенты проверяют токены криптографически без сетевых обращений в БД сервиса.
- **Ошибки и ограничения:** публичные ответы кэшируются через HTTP-заголовки `Cache-Control`. Приватные ключи надёжно защищены и не отдаются в JWKS.

### FederatedOidcLogin — федеративный вход через внешний Identity Provider

- **Вход и инициатор:** корпоративный пользователь, выбравший федеративный вход через внешний Identity Provider по протоколу OIDC.
- **Ход сценария:**
  1. Пользователь перенаправляется во внешний Identity Provider по протоколу OIDC Authorization Code Flow.
  2. После возврата с кодом `ExternalIntegrationAdapter` (адаптер для внешних интеграций) обращается к Token Endpoint внешнего IdP и получает внешний `id_token` и `access_token`.
  3. Адаптер валидирует подпись `id_token` по JWKS внешнего Identity Provider, проверяет `issuer` и срок действия.
  4. Извлекаются claims: `sub` (уникальный ID во внешнем IdP), `email`, `email_verified`, `preferred_username`, `name`.
  5. В транзакции Identity DB:
     - Проверяется таблица `external_identity_accounts` по индексу `(provider, subject = sub)`;
     - Если связка найдена: извлекается существующая `Identity`, обновляется `last_authenticated_at`;
     - Если связка отсутствует: выполняется поиск по подтверждённому email. Если пользователь найден — создаётся новая связка `external_identity_accounts`; если нет — создаётся новая `Identity` (`status = 'active'`), подтверждённый `IdentityEmail` и `IdentityProfile`;
     - Создаётся внутренняя `Session` Careeria и выпускается локальный набор токенов (Access Token и Refresh Token Careeria).
- **Результат:** пользователь аутентифицирован в Careeria с сохранением прозрачной связки с аккаунтом внешнего Identity Provider.
- **Ошибки и ограничения:** `Unauthenticated` при невалидной подписи или истёкшем токене внешнего IdP. При деактивации аккаунта во внешнем IdP доступ к Careeria прекращается при следующей синхронизации.

## Границы транзакций и безопасность

### Аутентификация, профиль токенов и ключи подписи

Identity Service выпускает Access Tokens строго по профилю **RFC 9068** (JSON Web Token Profile for OAuth 2.0 Access Tokens) с использованием современной асимметричной криптографии:

- **Алгоритм подписи:** `EdDSA` на эллиптической кривой `Ed25519` (высокая производительность проверки, устойчивость к side-channel атакам, компактный размер подписи 64 байта).
- **Заголовок токена (Header):**
  ```json
  {
    "typ": "at+jwt",
    "alg": "EdDSA",
    "kid": "identity-key-2026-v1"
  }
  ```
- **Полезная нагрузка (Payload):**
  ```json
  {
    "iss": "https://auth.careeria.ru",
    "sub": "a5e8c1b2-3d4f-4a6b-8c1d-2e3f4a5b6c7d",
    "aud": ["careeria-api"],
    "client_id": "careeria-web",
    "exp": 1789564800,
    "iat": 1789563900,
    "jti": "d8e7f6a5-b4c3-2d1e-0f9a-8b7c6d5e4f3a",
    "sid": "f1e2d3c4-b5a6-7890-1234-56789abcdef0",
    "scope": "openid profile email",
    "acr": "single_factor",
    "amr": ["password"],
    "auth_time": 1789563900
  }
  ```
- **Локальная валидация на шлюзе (API Gateway):** API Gateway кэширует публичный JWKS сервиса (`/.well-known/jwks.json`) и проверяет валидность токена криптографически без синхронных RPC-вызовов к Identity DB на каждый HTTP-запрос.

### Безопасность учётных данных и защита от подбора

Для защиты пользовательских паролей от перебора на GPU/ASIC используется алгоритм **Argon2id** (победитель Password Hashing Competition), сочетающий защиту от атак по побочным каналам (Argon2i) и атак с оптимизацией памяти (Argon2d):

- **Параметры хеширования:**
  - Время (итерации, t): `3`
  - Память (m): `64 МБ` (65 536 КБ)
  - Параллелизм (p): `4`
  - Длина соли: `16 байт` (криптографический `crypto/rand`)
  - Длина результирующего ключа: `32 байта`
- **Формат хранения (PHC String Format):**
  `argon2idv=19m=65536,t=3,p=4<base64-salt><base64-hash>`
- **Политика защиты от подбора (Brute-Force Lockout):**
  - Неуспешные попытки фиксируются в таблице `identity_password_auth_state`.
  - После 5 подряд неудачных попыток аккаунт блокируется на 15 минут (`locked_until = now + 15m`).
  - При попытке входа в заблокированный аккаунт проверка хеша не выполняется, что экономит вычислительные ресурсы CPU.

### Внешняя федеративная аутентификация (OIDC)

Careeria поддерживает гибридную схему аутентификации:
1. Локальная база пользователей платформы (соискатели, рекрутеры небольших компаний) аутентифицируется через Identity Service с локальными паролями Argon2id.
2. Корпоративные клиенты и партнёры используют внешний Identity Provider (Keycloak SSO) по протоколу OpenID Connect:
   - Взаимодействие изолировано в адаптере для внешних интеграций (`External Integration Adapter`).
   - Внешний субъект (`sub`) связывается с локальной identity через таблицу `external_identity_accounts(provider, subject)`.
   - Внешний email автоматически помечается как подтверждённый, если внешний IdP подтвердил `email_verified: true`.
   - При изменении корпоративного статуса пользователя во внешнем IdP при следующем входе сессия актуализируется, а при деактивации во внешнем IdP — доступ к Careeria блокируется.

### Транзакции, outbox и Debezium CDC

Все изменения, имеющие значение для других сервисов платформы и аудита безопасности, фиксируются компонентом Transactional Outbox в таблице `outbox` в той же SQL-транзакции PostgreSQL, что и изменение доменного состояния (Identity, Session, Token Family).

В сервисе **нет отдельного outbox-воркера**: фоновый поллинг БД не выполняется. Платформенный коннектор Debezium CDC считывает события напрямую из журнала упреждающей записи PostgreSQL (WAL) и надёжно публикует их в топик Kafka `identity.events` с гарантией At-Least-Once. Это исключает накладные расходы на периодические SELECT-запросы и конкурентные блокировки строк в рантайме сервиса.

| Событие | Назначение / триггер | Содержимое полезной нагрузки (Payload) |
|---|---|---|
| `IdentityCreated` | Первичная регистрация пользователя | `identity_id`, `status = 'pending'`, `created_at` |
| `IdentityVerified` | Успешное подтверждение email/телефона | `identity_id`, `verified_contact`, `contact_type`, `verified_at` |
| `IdentityProfileUpdated` | Обновление ФИО / персональных данных | `identity_id`, `version`, `family_name`, `given_name`, `updated_at` |
| `SessionCreated` | Успешная аутентификация и выпуск сессии | `session_id`, `identity_id`, `client_id`, `auth_level`, `created_at` |
| `TokenRotated` | Успешная ротация refresh-токена | `family_id`, `session_id`, `previous_token_id`, `new_token_id`, `rotated_at` |
| `RefreshTokenReuseDetected` | Инцидент безопасности: попытка повторного использования токена | `family_id`, `session_id`, `reused_token_id`, `detected_at`, `revoked_at` |
| `SessionRevoked` | Отзыв сессии (logout, блокировка, reuse) | `session_id`, `identity_id`, `reason`, `revoked_at` |
| `IdentityStatusChanged` | Блокировка или удаление аккаунта | `identity_id`, `previous_status`, `new_status`, `reason`, `updated_at` |

### Ошибки, конкуренция и наблюдаемость

- **Модель ошибок:** доменные ошибки строго типизированы (`apperr.Type`) и маппятся в стандартные gRPC Status Codes:
  - `InvalidArgument` (`registration_invalid_input`, `invalid_phone_format`);
  - `NotFound` (`identity_not_found`, `transaction_not_found`);
  - `AlreadyExists` (`registration_contact_already_exists`);
  - `Unauthenticated` (`invalid_credentials`, `token_expired`, `invalid_grant`);
  - `PermissionDenied` (`account_locked`, `account_disabled`);
  - `FailedPrecondition` / `Conflict` (`refresh_token_reuse_detected`, `invalid_step`).
- **Метрики (Prometheus):**
  - `identity_registration_total{status}` — количество регистраций;
  - `identity_authentication_total{method, result}` — количество попыток входа и успехов/неудач;
  - `identity_token_exchange_total{grant_type, result}` — интенсивность обмена токенов;
  - `identity_refresh_token_reuse_detected_total` — критический счётчик атак повторного использования токенов;
  - `identity_argon2id_duration_seconds` — гистограмма времени вычисления хеша Argon2id;
  - `debezium_streaming_milli_seconds_behind_source` — отставание CDC коннектора при репликации outbox в Kafka.
- **Трассировка и аудит:** сквозной `request_id` передаётся через OpenTelemetry контекст. Каждое критическое действие фиксируется в немодифицируемой таблице `security_events`.

## Что требуется закрепить перед реализацией

| Вопрос | Предлагаемое решение / следующий этап проектирования |
|---|---|
| Полное удаление учётной записи | Текущий сценарий выполняет мягкое удаление. Полная физическая очистка данных требует координации распределённой саги с Candidate, Company, Hiring и File Service. |
| Аппаратные ключи WebAuthn / Passkeys | В архитектуре заложены поля `authentication_level = multi_factor` и расширяемый enum методов входа, но сам FIDO2/WebAuthn протокол будет поддержан в следующем релизе. |
| Управление сложными корпоративными правами | Identity Service отвечает строго за аутентификацию и выпуск базовых claims; разграничение прав доступа к вакансиям и кандидатам вынесено в OpenFGA. |
| Ротация приватных ключей подписи | Поддерживается двухфазная ротация; автоматический шедулер выпуска ключей по расписанию конфигурируется эксплуатацией через Admin API. |

---

[Все L3](../README.md) · [L2 платформенных сервисов](../../projections/members/member4/member4-l2-containers.md) · [Integration Service](integration-service.md) · [Notification Service](notification-service.md) · [Company Service](company-service.md)
