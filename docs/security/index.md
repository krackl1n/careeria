# Безопасность и доступ

Целевые правила аутентификации, авторизации и защиты интеграций описаны в [архитектурной документации](../architecture/c4-plant-uml/README.md).

Платформенные сервисы доступа раскрыты в [области Member 4](../architecture/c4-plant-uml/plantuml/projections/members/member4/README.md).

Makefile содержит команду проверки модели OpenFGA и тестов авторизации:

```sh
make authz-model-test
```

Она использует Docker и файлы модели в `authz/openfga`. Наличие и актуальность этих файлов следует проверять в используемой версии проекта.
