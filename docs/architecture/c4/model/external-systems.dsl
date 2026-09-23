identityProvider = softwareSystem "Identity Provider" {
    description "Внешний провайдер аутентификации по OAuth 2.0 и OpenID Connect."
    tags "ExternalSystem"
}

notificationProviders = softwareSystem "Notification Providers" {
    description "Внешние сервисы доставки электронной почты, push- и SMS-уведомлений."
    tags "ExternalSystem"
}

employerHrSystem = softwareSystem "Employer HR System" {
    description "Внешняя корпоративная система работодателя, которая обменивается с Careeria данными о кандидатах и результатах найма."
    tags "ExternalSystem"
}
