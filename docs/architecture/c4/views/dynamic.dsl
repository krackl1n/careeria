dynamic platform "C4-Dynamic-HRIntegration" {
    title "Careeria — обработка webhook от HR-системы"
    description "Приём изменения из внешней HR-системы и передача его владельцу доменных данных."

    employerHrSystem -> platform.apiGateway "Отправляет webhook"
    platform.apiGateway -> platform.integrationService "Передаёт webhook"
    platform.integrationService -> platform.kafka "Публикует событие"
    platform.kafka -> platform.hiringService "Доставляет событие"
    platform.hiringService -> platform.hiringDb "Сохраняет изменение"

    autoLayout lr 400 250
}

dynamic platform "C4-Dynamic-DomainEvent" {
    title "Careeria — доставка доменного события"
    description "Публикация события из transactional outbox и формирование пользовательского уведомления."

    platform.hiringService -> platform.hiringDb "Записывает данные и outbox"
    platform.hiringDb -> platform.debezium "Передаёт outbox"
    platform.debezium -> platform.kafka "Публикует событие"
    platform.kafka -> platform.notificationService "Доставляет событие"
    platform.notificationService -> notificationProviders "Отправляет уведомление"

    autoLayout lr 400 250
}
