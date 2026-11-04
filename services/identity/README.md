# Identity service

## Инициализация signing key и OAuth-клиента

После запуска Identity зарегистрируйте настроенный signing key, затем создайте публичный OAuth-клиент:

```bash
./scripts/identity/create-signing-key.sh
./scripts/identity/create-oauth-client.sh
```

Скрипты вызывают внутренние `SigningKeyAdminService` и `OAuthClientAdminService` и не подключаются к базе данных. Все параметры и примеры запуска описаны в [`scripts/identity/README.md`](../../scripts/identity/README.md).

Текущий `CreateSigningKey` не принимает параметры: он регистрирует публичные метаданные ключа, уже настроенного в Identity. Приватный ключ через admin API не передаётся.
