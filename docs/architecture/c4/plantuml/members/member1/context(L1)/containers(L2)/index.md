# Контейнеры области (L2)

Область Member 1: кандидаты и вакансии. Основные контейнеры: Candidate Service, Candidate DB, Vacancy Service и Vacancy DB.

Candidate DB хранит структурированный профиль, статусы и метаданные версий резюме, но не бинарные документы. Общий File Service управляет загрузкой, проверками и привязками файлов в Object Storage (MinIO / S3 API). Клиент загружает файл по ограниченному presigned URL; команды профиля и резюме проходят через Candidate Service. OpenFGA используется обоими бизнес-сервисами для ресурсных проверок доступа. Эти зависимости находятся внутри Careeria и переиспользуются из общей модели.

Общий вход в систему: `Пользователь → Web Application → BFF → API Gateway`.

```plantuml
@startuml
!include architecture/c4/plantuml/members/member1/context(L1)/containers(L2)/member1-l2-containers.puml
@enduml
```

[Контекст области](../index.md) · [Компоненты Candidate Service · L3](../../../../system-context%28L1%29/full-system%28L2%29/candidate-service%28L3%29/index.md) · [Компоненты Vacancy Service · L3](../../../../system-context%28L1%29/full-system%28L2%29/vacancy-service%28L3%29/index.md) · [Полная система](../../../../system-context%28L1%29/full-system%28L2%29/index.md)
