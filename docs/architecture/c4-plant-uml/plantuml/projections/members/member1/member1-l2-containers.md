# Контейнеры области

Область Member 1: кандидаты и вакансии. Основные контейнеры: Candidate Service, Candidate DB, Vacancy Service и Vacancy DB.

Candidate DB хранит структурированный профиль, статусы и метаданные версий резюме, но не бинарные документы. Общий File Service управляет загрузкой, проверками и привязками файлов в Object Storage (MinIO / S3 API). Клиент загружает файл по ограниченному presigned URL; команды профиля и резюме проходят через Candidate Service. OpenFGA используется обоими бизнес-сервисами для ресурсных проверок доступа. Эти зависимости находятся внутри Careeria и переиспользуются из общей модели.

Общий вход в систему: `Пользователь → Web Application → BFF → API Gateway`.

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/projections/members/member1/member1-l2-containers.puml
@enduml
```

[Контекст области](member1-l1-context.md) · [Компоненты Candidate Service · L3](../../../component/backend/candidate-service.md) · [Компоненты Vacancy Service · L3](../../../component/backend/vacancy-service.md) · [Командная интеграция](../../../integration/team.md)
