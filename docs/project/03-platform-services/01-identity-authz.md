# 3.1. Аутентификация и авторизация

## Identity Service

Отвечает за учётные записи пользователей и аутентификацию. Не хранит бизнес-данные кандидата или компании.

**Основные функции:**

- регистрация, вход и восстановление доступа;
- хранение credentials (Argon2id) и интеграция с внешним Identity Provider (**Keycloak**) по OAuth 2.0 / OIDC;
- выпуск и ротация access / refresh токенов (JWT);
- управление активными пользовательскими сессиями.

---

## Авторизация: OpenFGA и модель ReBAC

Для разграничения прав доступа на уровне конкретных объектов используется **OpenFGA** (Relationship-Based Access Control). OpenFGA выступает единым источником истины для отношений доступа:

```text
user:123         recruiter   company:42
vacancy:501      parent      company:42
application:900  parent      vacancy:501
interview:1001   parent      application:900
```

Это позволяет выполнять атомарные проверки вида:

- `can user:123 edit vacancy:501?`
- `can user:123 view application:900?`
- `can user:123 manage interview:1001?`

---

## AuthZ Synchronization / Projection Service

Обновляет отношения доступа в OpenFGA на основе доменных событий из Kafka и формирует read-optimized проекцию для быстрых проверок прав без лишних сетевых обращений:

```text
Domain Event (Kafka) ──▶ AuthZ Synchronization ──▶ OpenFGA / Read Projection
```

Проекция является производным состоянием и при необходимости может быть полностью перестроена. Подробнее о проверке модели доступа см. в разделе [Безопасность и доступ](../../security/index.md).

