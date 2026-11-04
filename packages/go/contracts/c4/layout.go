package c4

import (
	"bytes"
	"fmt"
	"strings"
)

type LayoutBuilder struct {
	model *ComponentModel
}

func NewLayoutBuilder(m *ComponentModel) *LayoutBuilder {
	return &LayoutBuilder{model: m}
}

func (b *LayoutBuilder) FindEnumOwner(enumName string) string {
	m := b.model
	for _, f := range m.Aggregate.Fields {
		if f.Type == enumName {
			return m.Aggregate.Name
		}
	}
	for _, ent := range m.Entities {
		for _, f := range ent.Fields {
			if f.Type == enumName {
				return ent.Name
			}
		}
	}
	for _, vo := range m.ValueObjs {
		for _, f := range vo.Fields {
			if f.Type == enumName {
				return vo.Name
			}
		}
	}
	for _, pr := range m.Projections {
		for _, f := range pr.Fields {
			if f.Type == enumName {
				return pr.Name
			}
		}
	}
	return m.Aggregate.Name
}

func (b *LayoutBuilder) BuildLayout() string {
	var buf bytes.Buffer
	m := b.model

	buf.WriteString("' ========================================================\n")
	buf.WriteString("' РАСПОЛОЖЕНИЕ И ВЫРАВНИВАНИЕ\n")
	buf.WriteString("' ========================================================\n\n")

	// 1. Внутри прикладного слоя
	if m.Registry != nil {
		buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.Service.Name, m.Registry.Name))
		if m.StrategyIface != nil {
			buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.Registry.Name, m.StrategyIface.Name))
			if len(m.StrategyImpls) > 0 {
				buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.StrategyIface.Name, m.StrategyImpls[0].Name))
				if len(m.StrategyImpls) > 1 {
					buf.WriteString(fmt.Sprintf("%s -[hidden]right-> %s\n", m.StrategyImpls[0].Name, m.StrategyImpls[1].Name))
				}
			}
		}
		buf.WriteString("\n")
	}

	// 2. Слой приложения над доменным слоем
	if len(m.StrategyImpls) > 0 && m.Aggregate.Name != "" {
		buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n\n", m.StrategyImpls[0].Name, m.Aggregate.Name))
	} else if m.Aggregate.Name != "" {
		buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n\n", m.Service.Name, m.Aggregate.Name))
	}

	// 3. Выравнивание элементов доменного слоя по колонкам (слева направо)
	rightmostDomainElement := m.Aggregate.Name
	if len(m.ExternalAggregates) > 0 {
		rightmostDomainElement = m.ExternalAggregates[len(m.ExternalAggregates)-1].Name
	} else if len(m.Projections) > 0 {
		rightmostDomainElement = m.Projections[len(m.Projections)-1].Name
	} else if len(m.Policies) > 0 {
		rightmostDomainElement = m.Policies[len(m.Policies)-1].Name
	}

	if len(m.Policies) > 0 {
		buf.WriteString(fmt.Sprintf("%s -[hidden]right-> %s\n", m.Aggregate.Name, m.Policies[0].Name))
		if len(m.ExternalAggregates) > 0 {
			buf.WriteString(fmt.Sprintf("%s -[hidden]right-> %s\n", m.Policies[0].Name, m.ExternalAggregates[0].Name))
		} else if len(m.Projections) > 0 {
			buf.WriteString(fmt.Sprintf("%s -[hidden]right-> %s\n", m.Policies[0].Name, m.Projections[0].Name))
		}
	}

	// 4. Внутри основного агрегата: вертикальная группировка сущностей и объектов-значений
	if m.Aggregate.Name != "" {
		if len(m.Entities) > 0 {
			buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.Aggregate.Name, m.Entities[0].Name))
			if len(m.ValueObjs) > 0 {
				buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.Entities[0].Name, m.ValueObjs[0].Name))
				for _, enum := range m.Enums {
					if b.FindEnumOwner(enum.Name) == m.ValueObjs[0].Name {
						buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.ValueObjs[0].Name, enum.Name))
					}
				}
			} else if len(m.Projections) > 0 && len(m.Policies) == 0 {
				buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.Entities[0].Name, m.Projections[0].Name))
			}

			if len(m.Entities) > 1 {
				buf.WriteString(fmt.Sprintf("%s -[hidden]right-> %s\n", m.Entities[0].Name, m.Entities[1].Name))
				for _, enum := range m.Enums {
					if b.FindEnumOwner(enum.Name) == m.Entities[1].Name {
						buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.Entities[1].Name, enum.Name))
					}
				}
			}
		} else if len(m.ValueObjs) > 0 {
			buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.Aggregate.Name, m.ValueObjs[0].Name))
			for _, enum := range m.Enums {
				if b.FindEnumOwner(enum.Name) == m.ValueObjs[0].Name {
					buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.ValueObjs[0].Name, enum.Name))
				}
			}
		}

		var aggEnums []EnumInfo
		for _, enum := range m.Enums {
			if b.FindEnumOwner(enum.Name) == m.Aggregate.Name {
				aggEnums = append(aggEnums, enum)
			}
		}
		if len(m.Entities) > 1 && len(aggEnums) > 0 {
			buf.WriteString(fmt.Sprintf("%s -[hidden]right-> %s\n", m.Entities[1].Name, aggEnums[0].Name))
		} else if len(m.Entities) == 1 && len(aggEnums) > 0 {
			buf.WriteString(fmt.Sprintf("%s -[hidden]right-> %s\n", m.Entities[0].Name, aggEnums[0].Name))
		} else if len(aggEnums) > 0 && m.Aggregate.Name != "" {
			buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.Aggregate.Name, aggEnums[0].Name))
		}
		for i := 0; i < len(aggEnums)-1; i++ {
			buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", aggEnums[i].Name, aggEnums[i+1].Name))
		}
		buf.WriteString("\n")
	}

	// 5. Внешние контракты выстраиваются вертикальным столбиком справа
	if len(m.Ports) > 0 {
		buf.WriteString(fmt.Sprintf("%s -[hidden]right-> %s\n", m.Service.Name, m.Ports[0].Name))
		for i := 0; i < len(m.Ports)-1; i++ {
			buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.Ports[i].Name, m.Ports[i+1].Name))
		}
		buf.WriteString("\n")

		if m.Aggregate.Name != "" {
			repoPortName := ""
			for _, port := range m.Ports {
				if strings.HasSuffix(port.Name, "Repository") {
					repoPortName = port.Name
					break
				}
			}
			if repoPortName != "" {
				buf.WriteString(fmt.Sprintf("%s -[hidden]right-> %s\n\n", rightmostDomainElement, repoPortName))
			}
		}
	}

	return buf.String()
}

