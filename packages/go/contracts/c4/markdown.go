package c4

import (
	"bytes"
	"fmt"
)

func GenerateMarkdown(m *ComponentModel) string {
	if m.Kind != "" && m.Kind != KindDomain {
		return generateLayeredMarkdown(m)
	}
	return generateDomainMarkdown(m)
}

func generateDomainMarkdown(m *ComponentModel) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("# %s — детализация кода L4\n\n", m.Config.Title))
	buf.WriteString(fmt.Sprintf("Этот раздел описывает внутреннее устройство, классы, доменные сущности, контракты и структуры компонента **%s**, входящего в контейнер `%s`.\n\n", m.Config.Name, m.Config.Container))
	buf.WriteString(fmt.Sprintf("> %s\n\n", m.Config.Description))

	buf.WriteString(fmt.Sprintf("![%s](./%s.svg)\n\n", m.Config.Title, m.Config.Slug))

	buf.WriteString("<details>\n<summary>PlantUML Исходный код</summary>\n\n")
	buf.WriteString("```plantuml\n")
	buf.WriteString("@startuml\n")
	buf.WriteString(fmt.Sprintf("!include %s.puml\n", m.Config.Slug))
	buf.WriteString("@enduml\n")
	buf.WriteString("```\n")
	buf.WriteString("</details>\n\n")

	buf.WriteString("## 1. Архитектурные границы и принципы\n\n")
	buf.WriteString("1. **Изоляция Bounded Context:** Компонент владеет исключительно своими агрегатами и доменной логикой.\n")
	buf.WriteString("2. **Чистая архитектура (Ports & Adapters):** Вся внешняя инфраструктура (база данных, другие сервисы, брокер) представлена исключительно через исходящие интерфейсы.\n")
	buf.WriteString("3. **Транзакционная атомарность:** Оркестрация сценариев мутации агрегатов выполняется в рамках транзакций через Unit of Work (`TransactionManager`).\n\n")

	buf.WriteString("## 2. Доменная модель (Domain Layer)\n\n")
	if m.Aggregate.Name != "" {
		buf.WriteString(fmt.Sprintf("### Корень агрегата `%s` (Aggregate Root)\n\n", m.Aggregate.Name))
		buf.WriteString("| Поле | Тип | Назначение |\n")
		buf.WriteString("|---|---|---|\n")
		for _, f := range m.Aggregate.Fields {
			buf.WriteString(fmt.Sprintf("| `%s` | `%s` | Доменное свойство агрегата |\n", f.Name, f.Type))
		}
		buf.WriteString("\n**Ключевые методы агрегата:**\n\n")
		for _, method := range m.Aggregate.Methods {
			if IsExported(method.Name) {
				buf.WriteString(fmt.Sprintf("* `+%s(%s): %s`\n", method.Name, method.Params, method.ReturnType))
			}
		}
		buf.WriteString("\n")
	}

	if len(m.Entities) > 0 {
		buf.WriteString("### Дочерние сущности (Entities)\n\n")
		for _, e := range m.Entities {
			buf.WriteString(fmt.Sprintf("#### Сущность `%s`\n\n", e.Name))
			for _, f := range e.Fields {
				buf.WriteString(fmt.Sprintf("* `%s: %s`\n", f.Name, f.Type))
			}
			buf.WriteString("\n")
		}
	}

	if len(m.ValueObjs) > 0 {
		buf.WriteString("### Объекты-значения (Value Objects)\n\n")
		for _, vo := range m.ValueObjs {
			buf.WriteString(fmt.Sprintf("#### Объект-значение `%s`\n\n", vo.Name))
			for _, f := range vo.Fields {
				buf.WriteString(fmt.Sprintf("* `%s: %s`\n", f.Name, f.Type))
			}
			buf.WriteString("\n")
		}
	}

	if len(m.Projections) > 0 {
		buf.WriteString("### Проекции чтения (Read Projections)\n\n")
		for _, proj := range m.Projections {
			buf.WriteString(fmt.Sprintf("#### Проекция `%s`\n\n", proj.Name))
			for _, f := range proj.Fields {
				buf.WriteString(fmt.Sprintf("* `%s: %s`\n", f.Name, f.Type))
			}
			buf.WriteString("\n")
		}
	}

	if len(m.ExternalAggregates) > 0 {
		buf.WriteString("### Внешние агрегаты (External Aggregates)\n\n")
		for _, ext := range m.ExternalAggregates {
			buf.WriteString(fmt.Sprintf("#### Агрегат `%s`\n\n", ext.Name))
			for _, f := range ext.Fields {
				buf.WriteString(fmt.Sprintf("* `%s: %s`\n", f.Name, f.Type))
			}
			buf.WriteString("\n")
		}
	}

	if len(m.Policies) > 0 {
		buf.WriteString("### Доменные политики (Domain Policies)\n\n")
		for _, policy := range m.Policies {
			buf.WriteString(fmt.Sprintf("#### Политика `%s`\n\n", policy.Name))
			for _, method := range policy.Methods {
				buf.WriteString(fmt.Sprintf("* `+%s(%s): %s`\n", method.Name, method.Params, method.ReturnType))
			}
			buf.WriteString("\n")
		}
	}

	buf.WriteString("## 3. Сценарии использования (Application Layer)\n\n")
	buf.WriteString(fmt.Sprintf("Прикладной сервис `%s` оркестрирует выполнение бизнес-сценариев:\n\n", m.Service.Name))
	for _, method := range m.Service.Methods {
		buf.WriteString(fmt.Sprintf("* `+%s(%s): %s`\n", method.Name, method.Params, method.ReturnType))
	}
	buf.WriteString("\n")

	if m.Registry != nil {
		buf.WriteString(fmt.Sprintf("### Стратегии прикладного уровня (%s)\n\n", m.Registry.Name))
		buf.WriteString(fmt.Sprintf("Для оркестрации стратегий используется объект стратегии `%s`.\n\n", m.Registry.Name))
		if m.StrategyIface != nil {
			buf.WriteString(fmt.Sprintf("Интерфейс стратегии `%s` реализуют:\n\n", m.StrategyIface.Name))
			for _, impl := range m.StrategyImpls {
				buf.WriteString(fmt.Sprintf("* `%s`\n", impl.Name))
			}
			buf.WriteString("\n")
		}
	}

	buf.WriteString("## 4. Внешние порты и контракты (Outbound Ports)\n\n")
	for _, port := range m.Ports {
		buf.WriteString(fmt.Sprintf("### Порт `%s`\n\n", port.Name))
		for _, method := range port.Methods {
			buf.WriteString(fmt.Sprintf("* `+%s(%s): %s`\n", method.Name, method.Params, method.ReturnType))
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

func generateLayeredMarkdown(m *ComponentModel) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("# %s — детализация кода L4\n\n", m.Config.Title))
	buf.WriteString(fmt.Sprintf("Этот раздел описывает внутреннее устройство, классы, контракты и структуры компонента **%s**, входящего в контейнер `%s`.\n\n", m.Config.Name, m.Config.Container))
	if m.Config.Description != "" {
		buf.WriteString(fmt.Sprintf("> %s\n\n", m.Config.Description))
	}

	buf.WriteString(fmt.Sprintf("![%s](./%s.svg)\n\n", m.Config.Title, m.Config.Slug))

	buf.WriteString("<details>\n<summary>PlantUML Исходный код</summary>\n\n")
	buf.WriteString("```plantuml\n")
	buf.WriteString("@startuml\n")
	buf.WriteString(fmt.Sprintf("!include %s.puml\n", m.Config.Slug))
	buf.WriteString("@enduml\n")
	buf.WriteString("```\n")
	buf.WriteString("</details>\n\n")

	buf.WriteString("## Архитектурные слои и компоненты\n\n")

	for i, layer := range m.Layers {
		buf.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, layer.Name))

		if len(layer.Classes) > 0 {
			buf.WriteString("#### Структуры и классы\n\n")
			for _, c := range layer.Classes {
				role := c.Role
				if role == "" {
					role = "Component"
				}
				buf.WriteString(fmt.Sprintf("##### `%s` <<%s>>\n\n", c.Name, role))

				if len(c.Fields) > 0 {
					buf.WriteString("| Поле / Зависимость | Тип |\n|---|---|\n")
					for _, f := range c.Fields {
						buf.WriteString(fmt.Sprintf("| `%s` | `%s` |\n", f.Name, f.Type))
					}
					buf.WriteString("\n")
				}

				if len(c.Methods) > 0 {
					buf.WriteString("| Метод | Сигнатура |\n|---|---|\n")
					for _, meth := range c.Methods {
						buf.WriteString(fmt.Sprintf("| `%s` | `(%s): %s` |\n", meth.Name, meth.Params, meth.ReturnType))
					}
					buf.WriteString("\n")
				}
			}
		}

		if len(layer.Interfaces) > 0 {
			buf.WriteString("#### Интерфейсы и контракты\n\n")
			for _, iface := range layer.Interfaces {
				buf.WriteString(fmt.Sprintf("##### `%s`\n\n", iface.Name))
				if len(iface.Methods) > 0 {
					buf.WriteString("| Метод | Сигнатура |\n|---|---|\n")
					for _, meth := range iface.Methods {
						buf.WriteString(fmt.Sprintf("| `%s` | `(%s): %s` |\n", meth.Name, meth.Params, meth.ReturnType))
					}
					buf.WriteString("\n")
				}
			}
		}
	}

	return buf.String()
}

