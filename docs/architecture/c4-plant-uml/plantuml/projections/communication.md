# Коммуникации и уведомления

Communication Service хранит сообщения в своей БД. File Service работает с вложениями через MinIO. Realtime Publisher публикует пользовательские сообщения в Centrifugo, браузер получает их через WebSocket. LiveKit обслуживает WebRTC-медиапотоки.

Notification Service обращается к внешним email, push и SMS-провайдерам. Источник событий для этих потоков показан в [event-driven проекции](event-driven.md).

```plantuml
@startuml
!include architecture/c4-plant-uml/plantuml/projections/communication.puml
@enduml
```
