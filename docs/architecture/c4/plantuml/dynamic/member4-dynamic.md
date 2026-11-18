# Identity Service — динамика регистрации пользователя

<!-- excerpt -->

При регистрации по электронной почте Identity Service создаёт учётную запись и запрос на подтверждение адреса. Веб-приложение получает следующий шаг регистрации через API Gateway.

Запрос формы проходит через серверную часть Next.js и API Gateway к RegisterIdentity. Identity загружает текущую транзакцию авторизации из Redis и создаёт учётную запись, профиль, парольные данные и запрос подтверждения адреса. В той же транзакции базы фиксируются события. Затем временный контекст обновляется, а веб-интерфейс получает следующий шаг регистрации.

```plantuml
@startuml
!include architecture/c4/plantuml/dynamic/member4-dynamic.puml
@enduml
```

<p align="center">Рисунок - Регистрация по электронной почте</p>

<p align="right">Таблица - Элементы. Регистрация по электронной почте</p>

| Элемент | Тип или технология | Назначение и ответственность |
|---|---|---|
| Пользователь | Пользователь | Создаёт учётную запись по email и паролю. |
| Web Application | Next.js, React, TypeScript, Tailwind CSS | Интерфейс React работает в браузере. Серверный BFF обрабатывает пользовательскую сессию и обращается к API Gateway. Страницы и BFF развёртываются вместе в одном приложении Next.js. |
| API Gateway | Go / grpc-gateway | API Gateway принимает HTTP-запрос от BFF, проверяет предъявленный токен по открытым ключам Identity и применяет политику доступа к методу. Доменный сервис получает вызов по gRPC. |
| Identity Service | Go / gRPC | Identity Service регистрирует пользователей, проверяет учётные данные, выпускает токены и управляет сессиями. Для входа через внешнего провайдера сервис поддерживает OIDC. |
| Identity DB | PostgreSQL | Учётные записи, профили, пароли, челленджи и outbox. |
| Redis | Redis / TTL | Хранение временной транзакции авторизации и её текущего шага. |

<p align="right">Таблица - Связи. Регистрация по электронной почте</p>

| Источник | Получатель | Содержание взаимодействия | Протокол или способ вызова |
|---|---|---|---|
| Пользователь | Web Application | 1-2: Отправляет форму регистрации. | UI / HTTPS |
| Web Application | API Gateway | 3: Передаёт регистрацию. | HTTP / JSON, порт 8080 |
| API Gateway | Identity Service | 4: RegistrationService.RegisterIdentity. Запрос и ответ передаются как сообщения Protobuf. | gRPC / HTTP2, порт 50052 |
| Identity Service | Redis | 5: Загружает транзакцию авторизации. | Redis RESP, порт 6379 |
| Identity Service | Identity DB | 6: Создаёт identity, профиль, пароль,. Получатель выполняет чтение или запись в базе данных сервиса. | Одна SQL-транзакция, порт 5432 |
| Identity Service | Redis | 7: Сохраняет контекст регистрации. | Redis RESP / TTL |
| Identity Service | API Gateway | 8: Возвращает RegisterIdentityResponse. Запрос и ответ передаются как сообщения Protobuf. | gRPC response |
| API Gateway | Web Application | 9: Передаёт результат регистрации. | HTTP 200 / JSON |
| Web Application | Пользователь | 10-11: Возвращает и показывает результат. | UI Render |

<!-- /excerpt -->

[Исходник PlantUML](member4-dynamic.puml)

## Участники и входные данные

<p align="right">Таблица - Участники и входные данные (Identity Service - динамика регистрации пользователя)</p>

| Участник | Роль |
|---|---|
| Пользователь и Web Application | Ввод регистрационных данных и отображение результата. |
| Серверный BFF приложения Next.js | Передача запроса веб-интерфейса в HTTP API шлюза. |
| API Gateway | Преобразование HTTP/JSON в gRPC-вызов Identity. |
| Identity Service | Проверка входных данных, создание учётной записи и возврат контекста регистрации. |
| Identity DB / PostgreSQL | Хранение identity, профиля, пароля, verification challenge и outbox. |
| Redis | Хранение временной транзакции авторизации и её текущего шага. |

На вход передаются ID уже существующей authorization transaction, email, пароль и ФИО. Предусловие — действующая транзакция на шаге `choose_authentication_method` и свободный email. Создание этой транзакции предшествует рассматриваемому запросу.

## Порядок взаимодействий

<p align="right">Таблица - Порядок взаимодействий (Identity Service - динамика регистрации пользователя)</p>

| Шаги | Действие |
|---|---|
| **1–4** | Пользователь отправляет форму. Запрос `POST /api/v1/registration` проходит через серверный BFF приложения Next.js и API Gateway. Шлюз вызывает `RegistrationService.RegisterIdentity` по gRPC `50052`. |
| **5** | Identity получает транзакцию авторизации из Redis и проверяет возможность регистрации. |
| **6** | Identity создаёт учётную запись со статусом `pending`, профиль, пароль с хешем Argon2id и email-челлендж. В той же SQL-транзакции сохраняются outbox-события `IdentityProfileUpdated` и `EmailVerificationChallengeIssued`. |
| **7** | Identity связывает транзакцию авторизации с созданной учётной записью и челленджем, сохраняет контекст в Redis со следующим шагом `email_verification`. |
| **8–10** | После успешного сохранения Identity возвращает `RegisterIdentityResponse` с контекстом регистрации через API Gateway и BFF в Web Application. |
| **11** | Интерфейс показывает пользователю результат регистрации. |

Связь 6 объединяет SQL-операции регистрации. Начало и завершение транзакции не выделены отдельными стрелками: диаграмма показывает взаимодействие контейнеров, а не трассировку SQL-команд. Внутренние чтения для формирования ответа также входят в работу Identity с PostgreSQL.

В коде обновление Redis выполняется внутри callback SQL-транзакции; успешный ответ формируется после её завершения. Redis остаётся самостоятельным хранилищем и не становится частью SQL-транзакции.

## Результат сценария

Учётная запись создана со статусом `pending`. В ответе содержится контекст со следующим шагом `email_verification`; сам код подтверждения через API не возвращается. Доставка кода и подтверждение email выполняются последующими сценариями.

Все стрелки имеют одинаковое оформление. Подписи указывают бизнес-действие и протокол взаимодействия.

## Соответствие коду

<p align="right">Таблица - Соответствие коду (Identity Service - динамика регистрации пользователя)</p>

| Часть сценария | Источник от корня репозитория |
|---|---|
| Регистрация и сохранение данных | `services/identity/internal/usecase/authentication/register_identity.go` |
| Подготовка ответа | `services/identity/internal/transport/grpc/registration/registration.go` |
| Создание verification challenge и outbox-события | `services/identity/internal/service/contactverification/start_email.go` |
| HTTP/RPC-контракт | `contracts/proto/careeria/identity/v1/registration_service.proto` |

[Развёртывание Identity](../members/member4/deployment/member4-deployment.md) · [Контейнеры Member 4](../members/member4/context%28L1%29/containers%28L2%29/index.md) · [Раздел Member 4](../members/member4/README.md)
