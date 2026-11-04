# Коммуникации и уведомления (L2)

Communication Service хранит сообщения в своей БД. File Service работает с вложениями через MinIO. Realtime Publisher публикует пользовательские сообщения в Centrifugo, браузер получает их через WebSocket. LiveKit обслуживает WebRTC-медиапотоки.

Notification Service обращается к внешним email, push и SMS-провайдерам. Источник событий для этих потоков показан в [event-driven проекции](../event-driven%28L2%29/index.md).

```plantuml
@startuml
!include architecture/c4/plantuml/communication(L2)/communication.puml
@enduml
```
