workspace "Careeria" "Архитектурная модель рекрутинговой системы Careeria" {

    !identifiers hierarchical

    model {
        !include model/people.dsl
        !include model/external-systems.dsl

        platform = softwareSystem "Careeria" {
            description "Рекрутинговая система полного цикла: от публикации вакансии и отклика до оценки, собеседования и найма кандидата."

            !include model/containers.dsl
        }

        !include model/relationships.dsl
    }

    views {
        !include views/context.dsl
        !include views/containers.dsl
        !include views/dynamic.dsl
        !include styles.dsl

        properties {
            "structurizr.sort" "created"
        }
    }

    configuration {
        scope softwaresystem
    }
}
