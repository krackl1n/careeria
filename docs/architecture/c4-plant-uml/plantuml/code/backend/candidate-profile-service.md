# Candidate Profile Service — детализация кода L4

Этот раздел описывает внутреннее устройство, классы, доменные сущности, контракты и стратегии компонента **Candidate Profile Service**, входящего в контейнер `Candidate Service`.

Диаграмма детализирует реализацию компонента на уровне кода (C4 Level 4), раскрывая устройство агрегата `CandidateProfile`, паттерн Стратегия для маскирования персональных данных под разные роли и контракты с внешними компонентами.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/code/backend/candidate-profile-service.puml
@enduml
```

## Архитектурные принципы компонента

1. **Изоляция в границах компонента (C4 L4):**
   * Внутренняя схема кода не содержит адаптеров внешних систем (`PostgresCandidateRepository`, `CandidateGRPCHandler`, `OpenFGA Client`), так как они вынесены в самостоятельные L3-компоненты.
   * Контракты взаимодействия объявлены в виде интерфейсов `CandidateProfileRepository`, `OutboxRepository`, `AuthorizationPort`.
2. **Идиоматичный сервис в Go:**
   * Сценарии работы с профилем не разбиты на десятки микроструктур, а объединены в связный прикладной сервис `CandidateProfileService`.
   * Каждое действие имеет типизированную входную модель (`CreateProfileInput`, `UpdateSkillsInput`, `ChangeStatusInput`, `GetProfileInput`), а запросы возвращают типизированную выходную модель `ProfileOutput`. Это исключает засорение сигнатур примитивными типами.
3. **Паттерн Стратегия в Доменном слое (`MaskingStrategy`):**
   * Правила проекции профиля под роли (`OWNER`, `RECRUITER`, `ANONYMOUS`) инкапсулированы в семейство доменных стратегий с реестром, исключая ветвление через `switch-case`.
   * Стратегии формируют выходную модель `ProfileOutput`.

---

## Доменная модель (Domain Layer)

### 1. Корень агрегата `CandidateProfile` (Aggregate Root)

Управляет профессиональным профилем соискателя, карьерными предпочтениями, навыками и статусом поиска работы.

* **Поля агрегата:**
  * `id: UUID` — уникальный идентификатор профиля.
  * `userID: UUID` — владелец профиля (пользователь платформы).
  * `status: SearchStatus` — статус поиска работы.
  * `skills: List<Skill>` — перечень подтвержденных профессиональных навыков соискателя.
* **Бизнес-методы агрегата:**
  * `SetSkills(skills: List<Skill>): error` — валидирует и обновляет список навыков кандидата.
  * `TransitionStatus(newStatus: SearchStatus): error` — выполняет переход статуса поиска работы с проверкой допустимости переходов (например, архивный профиль не может стать `ACTIVELY_LOOKING` без предварительного восстановления).

### 2. Паттерн Стратегия для маскирования данных

* **`MaskingStrategy` (интерфейс стратегии):**  
  Определяет единый контракт проекции агрегата в выходную модель: `Apply(profile: CandidateProfile): ProfileOutput`.
* **Конкретные стратегии:**
  * `OwnerMaskingStrategy` — отдает владельцу профиля полные данные, включая приватные поля и контакты.
  * `RecruiterMaskingStrategy` — фильтрует контактные данные согласно настройке видимости (`HIDDEN` / `AUTHORIZED_RECRUITERS`).
* **`StrategyRegistry` (реестр стратегий):**  
  Хранит соответствие ролей пользователей и стратегий (`map[ViewerRole]MaskingStrategy`), позволяя юзкейсу резолвить нужный алгоритм без ветвлений.

---

## Внешние контракты (Ports & Repositories)

| Интерфейс | Назначение | Реализующий L3-компонент |
|---|---|---|
| `CandidateProfileRepository` | Поиск и сохранение агрегата `CandidateProfile` в БД. | `Candidate Repo` (Go / pgx, sqlc) |
| `OutboxRepository` | Фиксация доменных событий изменения профиля для публикации через Transactional Outbox. | `Transactional Outbox` (Go / pgx, sqlc) |
| `AuthorizationPort` | Проверка прав доступа и определение роли запрашивающего пользователя. | `Authorization Client` (Go / OpenFGA SDK) |

