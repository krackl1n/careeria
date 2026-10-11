# 2.3. Оценка кандидатов и коммуникации

## Assessment Service

Отвечает за проверку профессиональных навыков кандидатов. Оценивает соискателя, но не меняет этап `Application` напрямую — вместо этого публикует доменные события (например, `AssessmentCompleted`) в Kafka, которые обрабатывает Hiring Service.

**Основные доменные сущности:**

- `Assessment` и `Assessment Template` — назначенная проверка и её шаблон;
- `Assessment Session` — сессия прохождения испытания с контролем времени;
- `Test` и `Question` — тестовые задания и вопросы;
- `Practical Task` — практические задачи с проверкой решения;
- `Live Coding Session` — интерактивная сессия написания кода в реальном времени;
- `Submission`, `Score`, `Evaluation` — отправленный ответ, набранный балл и итоговая рецензия.

```text
Assessment Service ──(AssessmentCompleted)──▶ Kafka ──▶ Hiring Service (State Machine)
```

---

## Communication Service

Отвечает за текстовое взаимодействие участников на протяжении всего цикла найма.

**Основные доменные сущности:**

- `Conversation` — диалог или групповой чат;
- `Participant` — участник беседы;
- `Message` — сообщение в диалоге;
- `Attachment` — ссылка (`file_id`) на прикреплённый файл в File Service;
- `Reaction` и `Read Receipt` — реакции и отметки о прочтении.

**Поддерживаемые сценарии:**

- личная переписка `Candidate ↔ Recruiter`;
- групповые чаты команды подбора;
- встроенный чат во время видеособеседования;
- мгновенная доставка событий (`message.created`, `message.read`, `typing.started`) через Centrifugo.

Схема контура коммуникаций: [Коммуникации и уведомления](../../architecture/c4-plant-uml/plantuml/projections/communication.md).

