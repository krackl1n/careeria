# Identity Service: функции и юзкейсы

Каталог разграничивает наличие кода и готовность сценария. [C4 L3](../../architecture/c4/plantuml/system-context%28L1%29/full-system%28L2%29/identity-service%28L3%29/index.md) описывает целевой сервис; [описание Identity](index.md) связывает его с репозиторием. Наличие handler или use case не является доказательством настройки внешнего HTTP-маршрута, развёртывания или прохождения сквозного теста.

```plantuml
@startuml
!include services/identity/diagrams/identity-use-cases.puml
@enduml
```

Диаграмма показывает группы целей, для которых найдены зарегистрированные gRPC handlers; подробная готовность и возможные ограничения каждого сценария требуют проверки. Пользователь управляет входом и согласием, OAuth-клиент инициирует авторизацию/обмен, администратор вызывает внутренние API настройки. Целевые federation, revoke и logout не изображены как уже подключённые внешние сценарии.

| Сценарий | Компонент L3 | Подтверждённый срез и ограничение |
|---|---|---|
| RegisterIdentity | Authentication & Registration | Зарегистрирован gRPC Registration handler; логика в `usecase/authentication`. Условия, интеграция клиента и маршруты требуют сквозного теста. |
| GetAuthenticationMethods / GetAuthenticationContext | Authentication & Registration | Найдены методы зарегистрированного Authentication handler. |
| AuthenticateWithPassword | Authentication & Registration | Найден handler и use case проверки пароля; политику блокировки и хранение следует проверять с конкретной конфигурацией. |
| VerifyEmail / VerifyPhone | Authentication & Registration | Найдены методы handler и use cases; доставка OTP и end-to-end повторы зависят от интеграции. |
| SkipMFAEnrollment | Authentication & Registration | Найден handler для пропуска настройки; наличие доменных MFA-факторов не подтверждает полный сценарий enrollment/challenge. |
| GrantAuthorizationConsent / DenyAuthorizationConsent | Authentication & Registration | Найдены методы handler и логика use case согласия/отказа. |
| Authorize / ContinueAuthorization | OAuth & Token | Найдены зарегистрированные gRPC handlers и use cases; корректность публичного OAuth-пути требует отдельного сквозного теста. |
| ExchangeToken: authorization_code / refresh_token | OAuth & Token | Handler явно маршрутизирует оба grant type в use cases. Повторное использование refresh-токена обрабатывается в коде; конкурентные гарантии проверяются с БД. |
| Discovery / JWKS | Token Minting & Key | Зарегистрирован IdentityMetadata handler для конфигурации и набора ключей. Текущая конфигурация discovery не доказывает доступность каждого объявленного адреса. |
| CreatePublicOAuthClient / CreateSigningKey | Admin | Зарегистрированы внутренние admin handlers. `CreateSigningKey` регистрирует метаданные уже настроенного ключа, не принимает приватный ключ через запрос. Защита и эксплуатационный процесс требуют проверки. |
| RevokeToken / GetUserInfo | OAuth & Token | Есть use cases и их unit-тесты; в просмотренном OAuth gRPC handler не найдены методы для этих операций. Доступность через другой маршрут не подтверждена. |
| GetIdentitySession / LogoutIdentitySession | OAuth & Token | Указаны на C4 L3, но отдельный зарегистрированный gRPC Session handler и соответствующие use cases при текущей проверке не найдены; требуется аудит контрактов и gateway. |
| FederatedOidcLogin | External Integration Adapter | Заявлен на C4 L3; есть модель внешних аккаунтов, но полный путь федерации в просмотренных usecase/transport/integrations не найден. Не считать реализованным без дополнительной проверки. |

Список не исчерпывает доменные модели и зарезервированные таблицы. Перед включением функции в внешний контракт необходимо проверить её регистрацию, авторизацию, интеграции и сквозные сценарии; ошибки и ревокации не следует выводить только из схемы БД.
