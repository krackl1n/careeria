# Identity Service — динамика регистрации пользователя

Диаграмма показывает один сценарий: **создание учётной записи по email и паролю через `RegisterIdentity`**. Он начинается с отправки формы и заканчивается отображением результата регистрации. Нумерация 1–11 задаёт порядок взаимодействий контейнеров L2.

```plantuml
@startuml
!include architecture/c4/plantuml/dynamic/member4-dynamic.puml
@enduml
```

[Исходник PlantUML](member4-dynamic.puml)

## Участники и входные данные

| Участник | Роль |
|---|---|
| Пользователь и Web Application | Ввод регистрационных данных и отображение результата. |
| BFF | Передача запроса веб-интерфейса в HTTP API шлюза. |
| API Gateway | Преобразование HTTP/JSON в gRPC-вызов Identity. |
| Identity Service | Проверка входных данных, создание учётной записи и возврат контекста регистрации. |
| Identity DB / PostgreSQL | Хранение identity, профиля, пароля, verification challenge и outbox. |
| Redis | Хранение временной транзакции авторизации и её текущего шага. |

На вход передаются ID уже существующей authorization transaction, email, пароль и ФИО. Предусловие — действующая транзакция на шаге `choose_authentication_method` и свободный email. Создание этой транзакции предшествует рассматриваемому запросу.

## Порядок взаимодействий

| Шаги | Действие |
|---|---|
| **1–4** | Пользователь отправляет форму. Запрос `POST /api/v1/registration` проходит через Web Application, BFF и API Gateway. Шлюз вызывает `RegistrationService.RegisterIdentity` по gRPC `50052`. |
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

| Часть сценария | Источник от корня репозитория |
|---|---|
| Регистрация и сохранение данных | `services/identity/internal/usecase/authentication/register_identity.go` |
| Подготовка ответа | `services/identity/internal/transport/grpc/registration/registration.go` |
| Создание verification challenge и outbox-события | `services/identity/internal/service/contactverification/start_email.go` |
| HTTP/RPC-контракт | `contracts/proto/careeria/identity/v1/registration_service.proto` |

[Развёртывание Identity](../members/member4/deployment/member4-deployment.md) · [Контейнеры Member 4](../members/member4/context%28L1%29/containers%28L2%29/index.md) · [Раздел Member 4](../members/member4/README.md)
