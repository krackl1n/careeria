identityProvider = softwareSystem "Identity Provider" {
    description "OIDC-провайдер на базе Keycloak для федерации и внешней аутентификации пользователей."
    tags "ExternalSystem"
}

notificationProviders = softwareSystem "Notification Providers" {
    description "SendGrid для email, Firebase Cloud Messaging для push и Twilio для SMS."
    tags "ExternalSystem"
}

employerHrSystem = softwareSystem "Employer HR System" {
    description "Внешняя корпоративная система работодателя, которая обменивается с Careeria данными о кандидатах и результатах найма."
    tags "ExternalSystem"
}
