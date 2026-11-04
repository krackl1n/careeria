package c4

import (
	"bytes"
	"fmt"
	"strings"
)

func GeneratePlantUML(m *ComponentModel) string {
	if m.Kind != "" && m.Kind != KindDomain {
		return generateLayeredPlantUML(m)
	}
	return generateDomainPlantUML(m)
}

func generateDomainPlantUML(m *ComponentModel) string {
	var buf bytes.Buffer
	layout := NewLayoutBuilder(m)

	buf.WriteString(fmt.Sprintf("@startuml %s\n", m.Config.Slug))
	buf.WriteString("skinparam classAttributeIconSize 0\n")
	buf.WriteString("skinparam roundcorner 6\n")
	buf.WriteString("skinparam shadowing false\n")
	buf.WriteString("skinparam ranksep 45\n")
	buf.WriteString("skinparam nodesep 60\n")
	buf.WriteString("skinparam packagePadding 15\n\n")

	buf.WriteString("skinparam class {\n")
	buf.WriteString("    BackgroundColor #FFFFFF\n")
	buf.WriteString("    ArrowColor #2C3E50\n")
	buf.WriteString("    BorderColor #34495E\n")
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf("package \"Component: %s\" {\n\n", m.Config.Title))

	// 1. Application Layer
	buf.WriteString("    ' ===================================================\n")
	buf.WriteString("    ' 1. ПРИКЛАДНОЙ СЛОЙ (ОРКЕСТРАЦИЯ I/O И ЮЗКЕЙСЫ)\n")
	buf.WriteString("    ' ===================================================\n")
	buf.WriteString("    package \"Application Layer\" {\n")
	buf.WriteString(fmt.Sprintf("        class %s <<Application Service>> {\n", m.Service.Name))
	for _, f := range m.Service.Fields {
		buf.WriteString(fmt.Sprintf("            - %s: %s\n", f.Name, f.Type))
	}
	for _, method := range m.Service.Methods {
		buf.WriteString(fmt.Sprintf("            + %s(%s): %s\n", method.Name, method.Params, method.ReturnType))
	}
	buf.WriteString("        }\n\n")

	if m.Registry != nil || m.StrategyIface != nil || len(m.StrategyImpls) > 0 {
		if m.Registry != nil {
			buf.WriteString(fmt.Sprintf("        class %s <<%s>> {\n", m.Registry.Name, m.Registry.Role))
			for _, f := range m.Registry.Fields {
				buf.WriteString(fmt.Sprintf("            - %s: %s\n", f.Name, f.Type))
			}
			for _, method := range m.Registry.Methods {
				buf.WriteString(fmt.Sprintf("            + %s(%s): %s\n", method.Name, method.Params, method.ReturnType))
			}
			buf.WriteString("        }\n\n")
		}

		if m.StrategyIface != nil {
			buf.WriteString(fmt.Sprintf("        interface %s <<Strategy>> {\n", m.StrategyIface.Name))
			for _, method := range m.StrategyIface.Methods {
				buf.WriteString(fmt.Sprintf("            + %s(%s): %s\n", method.Name, method.Params, method.ReturnType))
			}
			buf.WriteString("        }\n\n")
		}

		if len(m.StrategyImpls) > 0 {
			buf.WriteString("        together {\n")
			for _, impl := range m.StrategyImpls {
				buf.WriteString(fmt.Sprintf("            class %s <<%s>> {\n", impl.Name, impl.Role))
				for _, f := range impl.Fields {
					buf.WriteString(fmt.Sprintf("                - %s: %s\n", f.Name, f.Type))
				}
				for _, method := range impl.Methods {
					buf.WriteString(fmt.Sprintf("                + %s(%s): %s\n", method.Name, method.Params, method.ReturnType))
				}
				buf.WriteString("            }\n")
			}
			buf.WriteString("        }\n")
		}
	}
	buf.WriteString("    }\n\n")

	// 2. Domain Layer
	buf.WriteString("    ' ===================================================\n")
	buf.WriteString("    ' 2. ДОМЕННЫЙ СЛОЙ (ЧИСТАЯ ЛОГИКА БЕЗ I/O)\n")
	buf.WriteString("    ' ===================================================\n")
	buf.WriteString("    package \"Domain Layer\" {\n\n")

	// Aggregate Root
	if m.Aggregate.Name != "" {
		buf.WriteString(fmt.Sprintf("        class %s <<Aggregate Root>> {\n", m.Aggregate.Name))
		for _, f := range m.Aggregate.Fields {
			prefix := "-"
			if IsExported(f.Name) {
				prefix = "+"
			}
			buf.WriteString(fmt.Sprintf("            %s %s: %s\n", prefix, f.Name, f.Type))
		}
		for _, method := range m.Aggregate.Methods {
			prefix := "-"
			if IsExported(method.Name) {
				prefix = "+"
			}
			buf.WriteString(fmt.Sprintf("            %s %s(%s): %s\n", prefix, method.Name, method.Params, method.ReturnType))
		}
		buf.WriteString("        }\n\n")
	}

	// Entities
	for _, entity := range m.Entities {
		buf.WriteString(fmt.Sprintf("        class %s <<Entity>> {\n", entity.Name))
		for _, f := range entity.Fields {
			prefix := "-"
			if IsExported(f.Name) {
				prefix = "+"
			}
			buf.WriteString(fmt.Sprintf("            %s %s: %s\n", prefix, f.Name, f.Type))
		}
		for _, method := range entity.Methods {
			prefix := "-"
			if IsExported(method.Name) {
				prefix = "+"
			}
			buf.WriteString(fmt.Sprintf("            %s %s(%s): %s\n", prefix, method.Name, method.Params, method.ReturnType))
		}
		buf.WriteString("        }\n\n")
	}

	// Domain Policies
	for _, policy := range m.Policies {
		buf.WriteString(fmt.Sprintf("        class %s <<Domain Policy>> {\n", policy.Name))
		for _, method := range policy.Methods {
			buf.WriteString(fmt.Sprintf("            + %s(%s): %s\n", method.Name, method.Params, method.ReturnType))
		}
		buf.WriteString("        }\n\n")
	}

	// Enums
	for _, enum := range m.Enums {
		buf.WriteString(fmt.Sprintf("        enum %s <<enumeration>> {\n", enum.Name))
		for _, val := range enum.Values {
			buf.WriteString(fmt.Sprintf("            %s\n", val))
		}
		buf.WriteString("        }\n\n")
	}

	// Value Objects
	for _, vo := range m.ValueObjs {
		buf.WriteString(fmt.Sprintf("        class %s <<Value Object>> {\n", vo.Name))
		for _, f := range vo.Fields {
			buf.WriteString(fmt.Sprintf("            + %s: %s\n", f.Name, f.Type))
		}
		for _, method := range vo.Methods {
			buf.WriteString(fmt.Sprintf("            + %s(%s): %s\n", method.Name, method.Params, method.ReturnType))
		}
		buf.WriteString("        }\n\n")
	}

	// Read Projections
	for _, proj := range m.Projections {
		buf.WriteString(fmt.Sprintf("        class %s <<%s>> {\n", proj.Name, proj.Role))
		for _, f := range proj.Fields {
			prefix := "-"
			if IsExported(f.Name) {
				prefix = "+"
			}
			buf.WriteString(fmt.Sprintf("            %s %s: %s\n", prefix, f.Name, f.Type))
		}
		for _, method := range proj.Methods {
			prefix := "-"
			if IsExported(method.Name) {
				prefix = "+"
			}
			buf.WriteString(fmt.Sprintf("            %s %s(%s): %s\n", prefix, method.Name, method.Params, method.ReturnType))
		}
		buf.WriteString("        }\n\n")
	}

	// External Aggregates (cross-domain)
	for _, ext := range m.ExternalAggregates {
		buf.WriteString(fmt.Sprintf("        class %s <<%s>> {\n", ext.Name, ext.Role))
		for _, f := range ext.Fields {
			prefix := "-"
			if IsExported(f.Name) {
				prefix = "+"
			}
			buf.WriteString(fmt.Sprintf("            %s %s: %s\n", prefix, f.Name, f.Type))
		}
		for _, method := range ext.Methods {
			prefix := "-"
			if IsExported(method.Name) {
				prefix = "+"
			}
			buf.WriteString(fmt.Sprintf("            %s %s(%s): %s\n", prefix, method.Name, method.Params, method.ReturnType))
		}
		buf.WriteString("        }\n\n")
	}

	buf.WriteString("    }\n\n")

	// 3. External Contracts (Ports)
	buf.WriteString("    ' ===================================================\n")
	buf.WriteString("    ' 3. ВНЕШНИЕ КОНТРАКТЫ (ПОРТЫ И РЕПОЗИТОРИИ)\n")
	buf.WriteString("    ' ===================================================\n")
	buf.WriteString("    package \"External Contracts (Ports & Repositories)\" {\n")
	for _, port := range m.Ports {
		buf.WriteString(fmt.Sprintf("        interface %s <<interface>> {\n", port.Name))
		for _, method := range port.Methods {
			buf.WriteString(fmt.Sprintf("            + %s(%s): %s\n", method.Name, method.Params, method.ReturnType))
		}
		buf.WriteString("        }\n")
	}
	buf.WriteString("    }\n")
	buf.WriteString("}\n\n")

	// 4. Layout
	buf.WriteString(layout.BuildLayout())

	// 5. Relationships
	buf.WriteString("' ========================================================\n")
	buf.WriteString("' СВЯЗИ\n")
	buf.WriteString("' ========================================================\n\n")

	if m.Aggregate.Name != "" {
		buf.WriteString(fmt.Sprintf("%s ..> %s : commands mutation\n", m.Service.Name, m.Aggregate.Name))
	}

	firstPort := true
	for _, port := range m.Ports {
		usesPort := false
		for _, f := range m.Service.Fields {
			if strings.TrimPrefix(f.Type, "*") == port.Name {
				usesPort = true
				break
			}
		}
		if usesPort {
			arrow := "-->"
			if firstPort && m.Registry == nil {
				arrow = "-right->"
				firstPort = false
			}
			buf.WriteString(fmt.Sprintf("%s %s %s : uses\n", m.Service.Name, arrow, port.Name))
		}
	}

	for _, entity := range m.Entities {
		buf.WriteString(fmt.Sprintf("%s *-- \"*\" %s : owns\n", m.Aggregate.Name, entity.Name))
	}

	for _, vo := range m.ValueObjs {
		used := false
		for _, mtd := range m.Aggregate.Methods {
			if strings.Contains(mtd.Params, vo.Name) {
				buf.WriteString(fmt.Sprintf("%s ..> %s : uses\n", m.Aggregate.Name, vo.Name))
				used = true
				break
			}
		}
		if !used {
			for _, f := range m.Aggregate.Fields {
				if f.Type == vo.Name {
					buf.WriteString(fmt.Sprintf("%s *-- %s\n", m.Aggregate.Name, vo.Name))
					used = true
					break
				}
			}
		}
	}

	for _, proj := range m.Projections {
		for _, mtd := range m.Aggregate.Methods {
			if strings.Contains(mtd.ReturnType, proj.Name) {
				buf.WriteString(fmt.Sprintf("%s ..> %s : produces\n", m.Aggregate.Name, proj.Name))
				break
			}
		}
		for _, p := range m.Ports {
			for _, mtd := range p.Methods {
				if strings.Contains(mtd.ReturnType, proj.Name) {
					buf.WriteString(fmt.Sprintf("%s ..> %s : provides\n", p.Name, proj.Name))
					break
				}
			}
		}
	}

	for _, ext := range m.ExternalAggregates {
		for _, p := range m.Ports {
			for _, mtd := range p.Methods {
				if ContainsTypeName(mtd.ReturnType, ext.Name) {
					buf.WriteString(fmt.Sprintf("%s ..> %s : provides\n", p.Name, ext.Name))
					break
				}
			}
		}
	}

	for _, enum := range m.Enums {
		owner := layout.FindEnumOwner(enum.Name)
		buf.WriteString(fmt.Sprintf("%s *-- \"1\" %s\n", owner, enum.Name))
	}

	for _, policy := range m.Policies {
		buf.WriteString(fmt.Sprintf("%s --> %s : executes policy\n", m.Service.Name, policy.Name))
		evaluated := make(map[string]bool)
		for _, method := range policy.Methods {
			if ContainsTypeName(method.Params, m.Aggregate.Name) && !evaluated[m.Aggregate.Name] {
				buf.WriteString(fmt.Sprintf("%s ..> %s : evaluates\n", policy.Name, m.Aggregate.Name))
				evaluated[m.Aggregate.Name] = true
			}
			for _, ext := range m.ExternalAggregates {
				if ContainsTypeName(method.Params, ext.Name) && !evaluated[ext.Name] {
					buf.WriteString(fmt.Sprintf("%s ..> %s : evaluates\n", policy.Name, ext.Name))
					evaluated[ext.Name] = true
				}
			}
			for _, proj := range m.Projections {
				if ContainsTypeName(method.Params, proj.Name) && !evaluated[proj.Name] {
					buf.WriteString(fmt.Sprintf("%s ..> %s : evaluates\n", policy.Name, proj.Name))
					evaluated[proj.Name] = true
				}
			}
			for _, ent := range m.Entities {
				if ContainsTypeName(method.Params, ent.Name) && !evaluated[ent.Name] {
					buf.WriteString(fmt.Sprintf("%s ..> %s : evaluates\n", policy.Name, ent.Name))
					evaluated[ent.Name] = true
				}
			}
			for _, vo := range m.ValueObjs {
				if ContainsTypeName(method.Params, vo.Name) && !evaluated[vo.Name] {
					buf.WriteString(fmt.Sprintf("%s ..> %s : evaluates\n", policy.Name, vo.Name))
					evaluated[vo.Name] = true
				}
			}
		}
	}

	if m.Registry != nil {
		buf.WriteString(fmt.Sprintf("%s --> %s : delegates operations\n", m.Service.Name, m.Registry.Name))
		if m.StrategyIface != nil {
			buf.WriteString(fmt.Sprintf("%s o-- %s : registry\n", m.Registry.Name, m.StrategyIface.Name))
		}
	}

	if m.StrategyIface != nil {
		for _, impl := range m.StrategyImpls {
			buf.WriteString(fmt.Sprintf("%s ..|> %s : implements\n", impl.Name, m.StrategyIface.Name))
			for _, f := range impl.Fields {
				if strings.Contains(f.Type, "Client") {
					buf.WriteString(fmt.Sprintf("%s -right-> %s : uses client\n", impl.Name, f.Type))
				}
			}
		}
	}

	buf.WriteString("\n@enduml\n")
	return buf.String()
}

func generateLayeredPlantUML(m *ComponentModel) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("@startuml %s\n", m.Config.Slug))
	buf.WriteString("skinparam classAttributeIconSize 0\n")
	buf.WriteString("skinparam roundcorner 6\n")
	buf.WriteString("skinparam shadowing false\n")
	buf.WriteString("skinparam ranksep 45\n")
	buf.WriteString("skinparam nodesep 60\n")
	buf.WriteString("skinparam packagePadding 15\n\n")

	buf.WriteString("skinparam class {\n")
	buf.WriteString("    BackgroundColor #FFFFFF\n")
	buf.WriteString("    ArrowColor #2C3E50\n")
	buf.WriteString("    BorderColor #34495E\n")
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf("package \"Component: %s\" {\n\n", m.Config.Title))

	for _, layer := range m.Layers {
		buf.WriteString(fmt.Sprintf("    package \"%s\" {\n", layer.Name))
		for _, c := range layer.Classes {
			role := c.Role
			if role == "" {
				role = "Component"
			}
			buf.WriteString(fmt.Sprintf("        class %s <<%s>> {\n", c.Name, role))
			for _, f := range c.Fields {
				if f.Type != "" {
					buf.WriteString(fmt.Sprintf("            + %s: %s\n", f.Name, f.Type))
				} else {
					buf.WriteString(fmt.Sprintf("            + %s\n", f.Name))
				}
			}
			for _, meth := range c.Methods {
				buf.WriteString(fmt.Sprintf("            + %s(%s): %s\n", meth.Name, meth.Params, meth.ReturnType))
			}
			buf.WriteString("        }\n\n")
		}

		for _, iface := range layer.Interfaces {
			buf.WriteString(fmt.Sprintf("        interface %s <<interface>> {\n", iface.Name))
			for _, meth := range iface.Methods {
				buf.WriteString(fmt.Sprintf("            + %s(%s): %s\n", meth.Name, meth.Params, meth.ReturnType))
			}
			buf.WriteString("        }\n\n")
		}

		for _, enum := range layer.Enums {
			buf.WriteString(fmt.Sprintf("        enum %s <<enumeration>> {\n", enum.Name))
			for _, val := range enum.Values {
				buf.WriteString(fmt.Sprintf("            %s\n", val))
			}
			buf.WriteString("        }\n\n")
		}

		buf.WriteString("    }\n\n")
	}

	buf.WriteString("}\n\n")

	if len(m.LayoutRules) > 0 {
		buf.WriteString("' ========================================================\n")
		buf.WriteString("' РАСПОЛОЖЕНИЕ И ВЫРАВНИВАНИЕ\n")
		buf.WriteString("' ========================================================\n\n")
		for _, rule := range m.LayoutRules {
			buf.WriteString(fmt.Sprintf("%s -[hidden]%s-> %s\n", rule.From, rule.Direction, rule.To))
		}
		buf.WriteString("\n")
	}

	if len(m.Relations) > 0 {
		buf.WriteString("' ========================================================\n")
		buf.WriteString("' СВЯЗИ\n")
		buf.WriteString("' ========================================================\n\n")
		for _, rel := range m.Relations {
			if rel.Label != "" {
				buf.WriteString(fmt.Sprintf("%s %s %s : %s\n", rel.From, rel.Arrow, rel.To, rel.Label))
			} else {
				buf.WriteString(fmt.Sprintf("%s %s %s\n", rel.From, rel.Arrow, rel.To))
			}
		}
		buf.WriteString("\n")
	}

	buf.WriteString("@enduml\n")
	return buf.String()
}

