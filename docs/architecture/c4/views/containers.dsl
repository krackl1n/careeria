container platform "C4-L2-HighLevel" {
    title "Careeria — верхнеуровневый обзор контейнеров"
    description "Точки входа и основные доменные группы без баз данных и тяжёлой инфраструктуры."

    include candidate
    include recruiter
    include hiringManager
    include companyAdmin
    include identityProvider
    include notificationProviders
    include employerHrSystem

    include platform.web
    include platform.bff
    include platform.apiGateway
    include platform.candidateService
    include platform.companyService
    include platform.vacancyService
    include platform.hiringService
    include platform.assessmentService
    include platform.communicationService
    include platform.identityService
    include platform.openFga
    include platform.notificationService
    include platform.fileService
    include platform.integrationService

    exclude "relationship.tag==InfrastructureConnection"

    autoLayout tb 350 250
    default
}

container platform "C4-L2-Hiring-Domain" {
    title "Careeria — домен найма и рекрутинга"
    description "Контейнеры, обеспечивающие работу рекрутеров и нанимающих руководителей."

    include recruiter
    include hiringManager
    include companyAdmin
    include employerHrSystem

    include platform.web
    include platform.bff
    include platform.apiGateway
    include platform.companyService
    include platform.companyDb
    include platform.vacancyService
    include platform.vacancyDb
    include platform.hiringService
    include platform.hiringDb
    include platform.assessmentService
    include platform.assessmentDb
    include platform.livekit
    include platform.integrationService

    exclude "relationship.tag==InfrastructureConnection"

    autoLayout lr 400 250
}

container platform "C4-L2-Candidate-Domain" {
    title "Careeria — домен кандидатов и вакансий"
    description "Контейнеры, обеспечивающие профиль кандидата, поиск вакансий и работу с файлами."

    include candidate
    include identityProvider

    include platform.web
    include platform.bff
    include platform.apiGateway
    include platform.identityService
    include platform.identityDb
    include platform.candidateService
    include platform.candidateDb
    include platform.vacancyService
    include platform.vacancyDb
    include platform.fileService
    include platform.objectStorage

    exclude "relationship.tag==InfrastructureConnection"

    autoLayout lr 400 250
}

container platform "C4-L2-Communication" {
    title "Careeria — коммуникации и уведомления"
    description "Чаты, realtime-доставка, видеосвязь, файлы и внешние каналы уведомлений."

    include candidate
    include recruiter
    include notificationProviders

    include platform.web
    include platform.bff
    include platform.apiGateway
    include platform.communicationService
    include platform.communicationDb
    include platform.notificationService
    include platform.fileService
    include platform.objectStorage
    include platform.realtimePublisher
    include platform.centrifugo
    include platform.livekit

    exclude "relationship.tag==InfrastructureConnection"

    autoLayout lr 400 250
}

container platform "C4-L2-Event-Driven" {
    title "Careeria — асинхронное взаимодействие (Kafka и Debezium)"
    description "Публикация transactional outbox, доставка событий и синхронизация технических проекций."

    include employerHrSystem
    include notificationProviders

    include platform.candidateDb
    include platform.companyDb
    include platform.vacancyDb
    include platform.hiringDb
    include platform.assessmentDb
    include platform.communicationDb
    include platform.identityDb
    include platform.debezium
    include platform.kafka
    include platform.schemaRegistry
    include platform.hiringService
    include platform.notificationService
    include platform.realtimePublisher
    include platform.authzProjection
    include platform.openFga
    include platform.integrationService

    autoLayout tb 400 250
}

container platform "C4-L2-BigPicture" {
    title "Careeria — генеральный план контейнеров"
    description "Все контейнеры платформы; сквозные связи OpenFGA, CDC и Schema Registry скрыты для читаемости."

    include *
    exclude "relationship.tag==InfrastructureConnection"

    autoLayout lr 600 400
}

container platform "C4-L2-Previous" {
    title "Careeria — полная контейнерная диаграмма (предыдущая версия)"
    description "Master View со всеми контейнерами и связями, сохранённый для полного обзора и сравнения."
    include *
    autoLayout lr 500 300
}
