package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Components []ComponentConfig `yaml:"components"`
}

type ComponentConfig struct {
	Name           string   `yaml:"name"`
	Slug           string   `yaml:"slug"`
	Container      string   `yaml:"container"`
	Title          string   `yaml:"title"`
	Description    string   `yaml:"description"`
	UsecasePkg     string   `yaml:"usecase_pkg"`
	DomainPkg      string   `yaml:"domain_pkg"`
	PolicyPkg      string   `yaml:"policy_pkg"`
	Subpackages    []string `yaml:"subpackages"`
	CrossDomainPkg string   `yaml:"cross_domain_pkg"`
	ServiceName    string   `yaml:"service_name"`
	Aggregate      string   `yaml:"aggregate"`
	OutputDir      string   `yaml:"output_dir"`
}

type FieldInfo struct {
	Name string
	Type string
}

type MethodInfo struct {
	Name       string
	Params     string
	ReturnType string
}

type StructInfo struct {
	Name    string
	Role    string
	Fields  []FieldInfo
	Methods []MethodInfo
}

type InterfaceInfo struct {
	Name    string
	Methods []MethodInfo
}

type EnumInfo struct {
	Name   string
	Values []string
}

type ComponentModel struct {
	Config             ComponentConfig
	Service            StructInfo
	Ports              []InterfaceInfo
	Aggregate          StructInfo
	Entities           []StructInfo
	ValueObjs          []StructInfo
	Projections        []StructInfo
	ExternalAggregates []StructInfo
	Enums              []EnumInfo
	Policies           []StructInfo
	StrategyIface      *InterfaceInfo
	Registry           *StructInfo
	StrategyImpls      []StructInfo
}

func main() {
	configPath := flag.String("config", "docs/architecture/c4-l4.yaml", "path to c4-l4 config")
	flag.Parse()

	data, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read config: %v\n", err)
		os.Exit(1)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse config: %v\n", err)
		os.Exit(1)
	}

	for _, comp := range cfg.Components {
		fmt.Printf("Processing L4 component: %s (%s)...\n", comp.Name, comp.Slug)
		model, err := analyzeComponent(comp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error analyzing %s: %v\n", comp.Name, err)
			os.Exit(1)
		}

		pumlContent := generatePlantUML(model)
		pumlPath := filepath.Join(comp.OutputDir, comp.Slug+".puml")
		if err := os.WriteFile(pumlPath, []byte(pumlContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write puml: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  -> Generated %s\n", pumlPath)

		mdContent := generateMarkdown(model)
		mdPath := filepath.Join(comp.OutputDir, comp.Slug+".md")
		if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write md: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  -> Generated %s\n", mdPath)
	}

	fmt.Println("Done generating C4 L4 specifications!")
}

func analyzeComponent(cfg ComponentConfig) (*ComponentModel, error) {
	model := &ComponentModel{
		Config: cfg,
	}

	fset := token.NewFileSet()

	// 1. Parse Usecase package
	usecasePkgs, err := parser.ParseDir(fset, cfg.UsecasePkg, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse usecase pkg %s: %w", cfg.UsecasePkg, err)
	}

	var serviceStruct *StructInfo
	var serviceMethods []MethodInfo

	for _, pkg := range usecasePkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					switch t := typeSpec.Type.(type) {
					case *ast.StructType:
						if typeSpec.Name.Name == "Service" {
							if serviceStruct == nil {
								serviceName := cfg.ServiceName
								if serviceName == "" {
									serviceName = "Service"
								}
								serviceStruct = &StructInfo{
									Name: serviceName,
									Role: "Application Service",
								}
								for _, field := range t.Fields.List {
									fieldType := formatType(field.Type)
									for _, name := range field.Names {
										serviceStruct.Fields = append(serviceStruct.Fields, FieldInfo{
											Name: name.Name,
											Type: fieldType,
										})
									}
								}
							}
						}
					case *ast.InterfaceType:
						if strings.HasSuffix(typeSpec.Name.Name, "Repository") ||
							strings.HasSuffix(typeSpec.Name.Name, "Port") ||
							strings.HasSuffix(typeSpec.Name.Name, "Checker") {
							iface := InterfaceInfo{
								Name: typeSpec.Name.Name,
							}
							for _, method := range t.Methods.List {
								if funcType, ok := method.Type.(*ast.FuncType); ok {
									for _, name := range method.Names {
										iface.Methods = append(iface.Methods, MethodInfo{
											Name:       name.Name,
											Params:     formatParams(funcType.Params),
											ReturnType: formatResults(funcType.Results),
										})
									}
								}
							}
							model.Ports = append(model.Ports, iface)
						}
					}
				}
			}

			// Parse methods on Service
			for _, decl := range file.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
					continue
				}
				recvType := formatType(funcDecl.Recv.List[0].Type)
				if strings.Contains(recvType, "Service") && funcDecl.Name.IsExported() {
					serviceMethods = append(serviceMethods, MethodInfo{
						Name:       funcDecl.Name.Name,
						Params:     formatParams(funcDecl.Type.Params),
						ReturnType: formatResults(funcDecl.Type.Results),
					})
				}
			}
		}
	}

	if serviceStruct != nil {
		serviceStruct.Methods = serviceMethods
		sort.Slice(serviceStruct.Methods, func(i, j int) bool {
			return serviceStruct.Methods[i].Name < serviceStruct.Methods[j].Name
		})
		model.Service = *serviceStruct
	}

	// 2. Parse Domain package
	domainPkgs, err := parser.ParseDir(fset, cfg.DomainPkg, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse domain pkg %s: %w", cfg.DomainPkg, err)
	}

	var allDomainStructs = make(map[string]*StructInfo)
	var allDomainEnums = make(map[string]*EnumInfo)
	var allConstants []struct {
		Name string
		Type string
		Val  string
	}

	for _, pkg := range domainPkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				if genDecl.Tok == token.CONST {
					for _, spec := range genDecl.Specs {
						valueSpec, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}
						typeName := ""
						if valueSpec.Type != nil {
							typeName = formatType(valueSpec.Type)
						}
						for i, name := range valueSpec.Names {
							val := name.Name
							if len(valueSpec.Values) > i {
								if basicLit, ok := valueSpec.Values[i].(*ast.BasicLit); ok {
									val = strings.Trim(basicLit.Value, `"`)
								}
							}
							allConstants = append(allConstants, struct {
								Name string
								Type string
								Val  string
							}{
								Name: name.Name,
								Type: typeName,
								Val:  val,
							})
						}
					}
					continue
				}
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					typeName := typeSpec.Name.Name
					switch t := typeSpec.Type.(type) {
					case *ast.StructType:
						sInfo := &StructInfo{
							Name: typeName,
						}
						hasID := false
						for _, field := range t.Fields.List {
							fieldType := formatType(field.Type)
							if len(field.Names) == 0 {
								// embedded
								sInfo.Fields = append(sInfo.Fields, FieldInfo{
									Name: fieldType,
									Type: "",
								})
							} else {
								for _, name := range field.Names {
									sInfo.Fields = append(sInfo.Fields, FieldInfo{
										Name: name.Name,
										Type: fieldType,
									})
									if name.Name == "ID" || strings.HasSuffix(name.Name, "ID") || strings.HasSuffix(name.Name, "Id") {
										hasID = true
									}
								}
							}
						}

						if typeName == cfg.Aggregate {
							sInfo.Role = "Aggregate Root"
						} else if strings.HasSuffix(typeName, "Policy") {
							sInfo.Role = "Domain Policy"
						} else if strings.HasSuffix(typeName, "Summary") || strings.HasSuffix(typeName, "Projection") || strings.HasSuffix(typeName, "View") || strings.HasSuffix(typeName, "ReadModel") {
							sInfo.Role = "Read Projection"
						} else if hasID {
							sInfo.Role = "Entity"
						} else {
							sInfo.Role = "Value Object"
						}
						allDomainStructs[typeName] = sInfo

					case *ast.Ident:
						if t.Name == "string" || t.Name == "int" {
							if genDecl.Tok == token.TYPE {
								allDomainEnums[typeName] = &EnumInfo{Name: typeName}
							}
						}
					}
				}
			}

		}
	}

	// Pass 2: Parse methods on domain structs
	for _, pkg := range domainPkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
					continue
				}
				recvType := cleanRecvType(formatType(funcDecl.Recv.List[0].Type))
				if sInfo, ok := allDomainStructs[recvType]; ok {
					sInfo.Methods = append(sInfo.Methods, MethodInfo{
						Name:       funcDecl.Name.Name,
						Params:     formatParams(funcDecl.Type.Params),
						ReturnType: formatResults(funcDecl.Type.Results),
					})
				}
			}
		}
	}

	// Populate enum values from collected constants
	for _, c := range allConstants {
		if c.Type != "" {
			if enum, ok := allDomainEnums[c.Type]; ok {
				enum.Values = append(enum.Values, c.Val)
			}
		} else {
			for eName, enum := range allDomainEnums {
				if strings.HasPrefix(c.Name, eName) ||
					(strings.HasPrefix(c.Name, "Role") && eName == "ParticipantRole") ||
					(strings.HasPrefix(c.Name, "Type") && eName == "InterviewType") ||
					(strings.HasPrefix(c.Name, "Format") && eName == "MeetingFormat") ||
					(strings.HasPrefix(c.Name, "Status") && eName == "InterviewStatus") ||
					(strings.HasPrefix(c.Name, "Stage") && eName == "PipelineStage") ||
					(strings.HasPrefix(c.Name, "Currency") && eName == "Currency") {
					enum.Values = append(enum.Values, c.Val)
					break
				}
			}
		}
	}

	// Categorize domain types
	for name, sInfo := range allDomainStructs {
		if strings.HasSuffix(name, "Payload") || strings.HasSuffix(name, "Request") || strings.HasSuffix(name, "Result") || strings.HasSuffix(name, "Event") {
			continue
		}
		if name == cfg.Aggregate {
			model.Aggregate = *sInfo
		} else if sInfo.Role == "Entity" {
			model.Entities = append(model.Entities, *sInfo)
		} else if sInfo.Role == "Domain Policy" {
			model.Policies = append(model.Policies, *sInfo)
		} else if sInfo.Role == "Read Projection" {
			model.Projections = append(model.Projections, *sInfo)
		} else {
			model.ValueObjs = append(model.ValueObjs, *sInfo)
		}
	}

	// 3. Parse Policies if in separate pkg
	if cfg.PolicyPkg != "" {
		policyPkgs, err := parser.ParseDir(fset, cfg.PolicyPkg, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, parser.ParseComments)
		if err == nil {
			for _, pkg := range policyPkgs {
				for _, file := range pkg.Files {
					for _, decl := range file.Decls {
						genDecl, ok := decl.(*ast.GenDecl)
						if !ok {
							continue
						}
						for _, spec := range genDecl.Specs {
							typeSpec, ok := spec.(*ast.TypeSpec)
							if !ok {
								continue
							}
							if _, ok := typeSpec.Type.(*ast.StructType); ok && strings.HasSuffix(typeSpec.Name.Name, "Policy") {
								pInfo := StructInfo{
									Name: typeSpec.Name.Name,
									Role: "Domain Policy",
								}
								model.Policies = append(model.Policies, pInfo)
							}
						}
					}

					for _, decl := range file.Decls {
						funcDecl, ok := decl.(*ast.FuncDecl)
						if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
							continue
						}
						recvType := cleanRecvType(formatType(funcDecl.Recv.List[0].Type))
						for i := range model.Policies {
							if model.Policies[i].Name == recvType {
								model.Policies[i].Methods = append(model.Policies[i].Methods, MethodInfo{
									Name:       funcDecl.Name.Name,
									Params:     formatParams(funcDecl.Type.Params),
									ReturnType: formatResults(funcDecl.Type.Results),
								})
							}
						}
					}
				}
			}
		}
	}

	// 4. Parse Subpackages (e.g. strategies / room)
	for _, subpkg := range cfg.Subpackages {
		subPkgs, err := parser.ParseDir(fset, subpkg, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, parser.ParseComments)
		if err == nil {
			for _, pkg := range subPkgs {
				for _, file := range pkg.Files {
					for _, decl := range file.Decls {
						genDecl, ok := decl.(*ast.GenDecl)
						if !ok {
							continue
						}
						for _, spec := range genDecl.Specs {
							typeSpec, ok := spec.(*ast.TypeSpec)
							if !ok {
								continue
							}
							switch t := typeSpec.Type.(type) {
							case *ast.InterfaceType:
								if strings.HasSuffix(typeSpec.Name.Name, "Client") {
									iface := InterfaceInfo{Name: typeSpec.Name.Name}
									for _, method := range t.Methods.List {
										if funcType, ok := method.Type.(*ast.FuncType); ok {
											for _, name := range method.Names {
												iface.Methods = append(iface.Methods, MethodInfo{
													Name:       name.Name,
													Params:     formatParams(funcType.Params),
													ReturnType: formatResults(funcType.Results),
												})
											}
										}
									}
									model.Ports = append(model.Ports, iface)
								} else if strings.HasSuffix(typeSpec.Name.Name, "Strategy") {
									iface := InterfaceInfo{Name: typeSpec.Name.Name}
									for _, method := range t.Methods.List {
										if funcType, ok := method.Type.(*ast.FuncType); ok {
											for _, name := range method.Names {
												iface.Methods = append(iface.Methods, MethodInfo{
													Name:       name.Name,
													Params:     formatParams(funcType.Params),
													ReturnType: formatResults(funcType.Results),
												})
											}
										}
									}
									model.StrategyIface = &iface
								}
							case *ast.StructType:
								if strings.HasSuffix(typeSpec.Name.Name, "Request") || strings.HasSuffix(typeSpec.Name.Name, "Result") || !isExported(typeSpec.Name.Name) {
									continue
								}
								if typeSpec.Name.Name == "RoomStrategy" {
									reg := StructInfo{
										Name: "RoomStrategy",
										Role: "Strategy",
										Fields: []FieldInfo{
											{Name: "registry", Type: "Map<MeetingFormat, MeetingRoomStrategy>"},
										},
									}
									model.Registry = &reg
								} else if strings.HasSuffix(typeSpec.Name.Name, "Strategy") {
									st := StructInfo{
										Name: typeSpec.Name.Name,
										Role: "Strategy",
									}
									for _, field := range t.Fields.List {
										fieldType := formatType(field.Type)
										for _, name := range field.Names {
											st.Fields = append(st.Fields, FieldInfo{
												Name: name.Name,
												Type: fieldType,
											})
										}
									}
									model.StrategyImpls = append(model.StrategyImpls, st)
								}
							}
						}
					}

					for _, decl := range file.Decls {
						funcDecl, ok := decl.(*ast.FuncDecl)
						if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
							continue
						}
						recvType := cleanRecvType(formatType(funcDecl.Recv.List[0].Type))
						if model.Registry != nil && model.Registry.Name == recvType && funcDecl.Name.IsExported() {
							model.Registry.Methods = append(model.Registry.Methods, MethodInfo{
								Name:       funcDecl.Name.Name,
								Params:     formatParams(funcDecl.Type.Params),
								ReturnType: formatResults(funcDecl.Type.Results),
							})
						}
						for i := range model.StrategyImpls {
							if model.StrategyImpls[i].Name == recvType && funcDecl.Name.IsExported() {
								model.StrategyImpls[i].Methods = append(model.StrategyImpls[i].Methods, MethodInfo{
									Name:       funcDecl.Name.Name,
									Params:     formatParams(funcDecl.Type.Params),
									ReturnType: formatResults(funcDecl.Type.Results),
								})
							}
						}
					}
				}
			}
		}
	}

	// 5. Parse Cross-Domain Package (e.g. read projections referenced by ports/policies)
	if cfg.CrossDomainPkg != "" {
		crossPkgs, err := parser.ParseDir(fset, cfg.CrossDomainPkg, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, parser.ParseComments)
		if err == nil {
			for _, pkg := range crossPkgs {
				for _, file := range pkg.Files {
					for _, decl := range file.Decls {
						genDecl, ok := decl.(*ast.GenDecl)
						if !ok {
							continue
						}
						for _, spec := range genDecl.Specs {
							typeSpec, ok := spec.(*ast.TypeSpec)
							if !ok {
								continue
							}
							typeName := typeSpec.Name.Name
							if t, ok := typeSpec.Type.(*ast.StructType); ok {
								isReferenced := false
								for _, port := range model.Ports {
									for _, m := range port.Methods {
										if containsTypeName(m.ReturnType, typeName) || containsTypeName(m.Params, typeName) {
											isReferenced = true
											break
										}
									}
									if isReferenced {
										break
									}
								}
								if !isReferenced {
									for _, pol := range model.Policies {
										for _, m := range pol.Methods {
											if containsTypeName(m.Params, typeName) {
												isReferenced = true
												break
											}
										}
										if isReferenced {
											break
										}
									}
								}

								if isReferenced {
									hasID := false
									for _, field := range t.Fields.List {
										for _, name := range field.Names {
											if name.Name == "ID" || strings.HasSuffix(name.Name, "ID") {
												hasID = true
											}
										}
									}
									role := "Aggregate Root"
									if strings.HasSuffix(typeName, "Policy") {
										role = "Domain Policy"
									} else if strings.HasSuffix(typeName, "Summary") || strings.HasSuffix(typeName, "Projection") {
										role = "Read Projection"
									} else if !hasID {
										role = "Value Object"
									}
									extInfo := StructInfo{
										Name: typeName,
										Role: role,
									}
									for _, field := range t.Fields.List {
										fieldType := formatType(field.Type)
										for _, name := range field.Names {
											extInfo.Fields = append(extInfo.Fields, FieldInfo{
												Name: name.Name,
												Type: fieldType,
											})
										}
									}
									model.ExternalAggregates = append(model.ExternalAggregates, extInfo)
								}
							}
						}
					}
				}
			}
		}
	}

	if model.Registry != nil {
		sort.Slice(model.Registry.Methods, func(i, j int) bool {
			return model.Registry.Methods[i].Name < model.Registry.Methods[j].Name
		})
	}
	hasClient := func(s StructInfo) bool {
		for _, f := range s.Fields {
			if strings.Contains(f.Type, "Client") {
				return true
			}
		}
		return false
	}
	sort.Slice(model.StrategyImpls, func(i, j int) bool {
		cI := hasClient(model.StrategyImpls[i])
		cJ := hasClient(model.StrategyImpls[j])
		if cI != cJ {
			return !cI
		}
		return model.StrategyImpls[i].Name < model.StrategyImpls[j].Name
	})
	for i := range model.StrategyImpls {
		sort.Slice(model.StrategyImpls[i].Methods, func(a, b int) bool {
			return model.StrategyImpls[i].Methods[a].Name < model.StrategyImpls[i].Methods[b].Name
		})
	}

	for _, enum := range allDomainEnums {
		model.Enums = append(model.Enums, *enum)
	}

	sort.Slice(model.Ports, func(i, j int) bool { return model.Ports[i].Name < model.Ports[j].Name })
	sort.Slice(model.Entities, func(i, j int) bool { return model.Entities[i].Name < model.Entities[j].Name })
	sort.Slice(model.ValueObjs, func(i, j int) bool { return model.ValueObjs[i].Name < model.ValueObjs[j].Name })
	sort.Slice(model.Projections, func(i, j int) bool { return model.Projections[i].Name < model.Projections[j].Name })
	sort.Slice(model.ExternalAggregates, func(i, j int) bool { return model.ExternalAggregates[i].Name < model.ExternalAggregates[j].Name })
	sort.Slice(model.Enums, func(i, j int) bool { return model.Enums[i].Name < model.Enums[j].Name })

	return model, nil
}

func generatePlantUML(m *ComponentModel) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("@startuml %s-l4\n", m.Config.Slug))
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
			if isExported(f.Name) {
				prefix = "+"
			}
			buf.WriteString(fmt.Sprintf("            %s %s: %s\n", prefix, f.Name, f.Type))
		}
		for _, method := range m.Aggregate.Methods {
			prefix := "-"
			if isExported(method.Name) {
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
			if isExported(f.Name) {
				prefix = "+"
			}
			buf.WriteString(fmt.Sprintf("            %s %s: %s\n", prefix, f.Name, f.Type))
		}
		for _, method := range entity.Methods {
			prefix := "-"
			if isExported(method.Name) {
				prefix = "+"
			}
			buf.WriteString(fmt.Sprintf("            %s %s(%s): %s\n", prefix, method.Name, method.Params, method.ReturnType))
		}
		buf.WriteString("        }\n\n")
	}

	// Strategies & Policies
	for _, policy := range m.Policies {
		buf.WriteString(fmt.Sprintf("        class %s <<Domain Policy>> {\n", policy.Name))
		for _, method := range policy.Methods {
			buf.WriteString(fmt.Sprintf("            + %s(%s): %s\n", method.Name, method.Params, method.ReturnType))
		}
		buf.WriteString("        }\n\n")
	}



	// Enums & Value Objects
	for _, enum := range m.Enums {
		buf.WriteString(fmt.Sprintf("        enum %s <<enumeration>> {\n", enum.Name))
		for _, val := range enum.Values {
			buf.WriteString(fmt.Sprintf("            %s\n", val))
		}
		buf.WriteString("        }\n\n")
	}

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

	for _, proj := range m.Projections {
		buf.WriteString(fmt.Sprintf("        class %s <<%s>> {\n", proj.Name, proj.Role))
		for _, f := range proj.Fields {
			prefix := "-"
			if isExported(f.Name) {
				prefix = "+"
			}
			buf.WriteString(fmt.Sprintf("            %s %s: %s\n", prefix, f.Name, f.Type))
		}
		for _, method := range proj.Methods {
			prefix := "-"
			if isExported(method.Name) {
				prefix = "+"
			}
			buf.WriteString(fmt.Sprintf("            %s %s(%s): %s\n", prefix, method.Name, method.Params, method.ReturnType))
		}
		buf.WriteString("        }\n\n")
	}

	for _, ext := range m.ExternalAggregates {
		buf.WriteString(fmt.Sprintf("        class %s <<%s>> {\n", ext.Name, ext.Role))
		for _, f := range ext.Fields {
			prefix := "-"
			if isExported(f.Name) {
				prefix = "+"
			}
			buf.WriteString(fmt.Sprintf("            %s %s: %s\n", prefix, f.Name, f.Type))
		}
		for _, method := range ext.Methods {
			prefix := "-"
			if isExported(method.Name) {
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

	// 4. Relationships & Layout
	buf.WriteString("' ========================================================\n")
	buf.WriteString("' РАСПОЛОЖЕНИЕ И ВЫРАВНИВАНИЕ\n")
	buf.WriteString("' ========================================================\n\n")

	// Внутри прикладного слоя
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

	// Слой приложения над доменным слоем
	if len(m.StrategyImpls) > 0 && m.Aggregate.Name != "" {
		buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n\n", m.StrategyImpls[0].Name, m.Aggregate.Name))
	} else if m.Aggregate.Name != "" {
		buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n\n", m.Service.Name, m.Aggregate.Name))
	}

	findOwner := func(enumName string) string {
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

	// Горизонтальное выравнивание политик и внешних агрегатов относительно основного агрегата
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

	// Внутри доменного слоя: компактная группировка
	if m.Aggregate.Name != "" {
		if len(m.Entities) > 0 {
			buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.Aggregate.Name, m.Entities[0].Name))
			if len(m.ValueObjs) > 0 {
				buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.Entities[0].Name, m.ValueObjs[0].Name))
				for _, enum := range m.Enums {
					if findOwner(enum.Name) == m.ValueObjs[0].Name {
						buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.ValueObjs[0].Name, enum.Name))
					}
				}
			} else if len(m.Projections) > 0 && len(m.Policies) == 0 {
				buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.Entities[0].Name, m.Projections[0].Name))
			}

			if len(m.Entities) > 1 {
				buf.WriteString(fmt.Sprintf("%s -[hidden]right-> %s\n", m.Entities[0].Name, m.Entities[1].Name))
				for _, enum := range m.Enums {
					if findOwner(enum.Name) == m.Entities[1].Name {
						buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.Entities[1].Name, enum.Name))
					}
				}
			}
		} else if len(m.ValueObjs) > 0 {
			buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.Aggregate.Name, m.ValueObjs[0].Name))
			for _, enum := range m.Enums {
				if findOwner(enum.Name) == m.ValueObjs[0].Name {
					buf.WriteString(fmt.Sprintf("%s -[hidden]down-> %s\n", m.ValueObjs[0].Name, enum.Name))
				}
			}
		}

		var aggEnums []EnumInfo
		for _, enum := range m.Enums {
			if findOwner(enum.Name) == m.Aggregate.Name {
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

	// Внешние контракты выстраиваются вертикальным столбиком справа
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
				if containsTypeName(mtd.ReturnType, ext.Name) {
					buf.WriteString(fmt.Sprintf("%s ..> %s : provides\n", p.Name, ext.Name))
					break
				}
			}
		}
	}

	for _, enum := range m.Enums {
		owner := findOwner(enum.Name)
		buf.WriteString(fmt.Sprintf("%s *-- \"1\" %s\n", owner, enum.Name))
	}

	for _, policy := range m.Policies {
		buf.WriteString(fmt.Sprintf("%s --> %s : executes policy\n", m.Service.Name, policy.Name))
		evaluated := make(map[string]bool)
		for _, method := range policy.Methods {
			if containsTypeName(method.Params, m.Aggregate.Name) && !evaluated[m.Aggregate.Name] {
				buf.WriteString(fmt.Sprintf("%s ..> %s : evaluates\n", policy.Name, m.Aggregate.Name))
				evaluated[m.Aggregate.Name] = true
			}
			for _, ext := range m.ExternalAggregates {
				if containsTypeName(method.Params, ext.Name) && !evaluated[ext.Name] {
					buf.WriteString(fmt.Sprintf("%s ..> %s : evaluates\n", policy.Name, ext.Name))
					evaluated[ext.Name] = true
				}
			}
			for _, proj := range m.Projections {
				if containsTypeName(method.Params, proj.Name) && !evaluated[proj.Name] {
					buf.WriteString(fmt.Sprintf("%s ..> %s : evaluates\n", policy.Name, proj.Name))
					evaluated[proj.Name] = true
				}
			}
			for _, ent := range m.Entities {
				if containsTypeName(method.Params, ent.Name) && !evaluated[ent.Name] {
					buf.WriteString(fmt.Sprintf("%s ..> %s : evaluates\n", policy.Name, ent.Name))
					evaluated[ent.Name] = true
				}
			}
			for _, vo := range m.ValueObjs {
				if containsTypeName(method.Params, vo.Name) && !evaluated[vo.Name] {
					buf.WriteString(fmt.Sprintf("%s ..> %s : evaluates\n", policy.Name, vo.Name))
					evaluated[vo.Name] = true
				}
			}
		}
	}

	if m.Registry != nil {
		buf.WriteString(fmt.Sprintf("%s --> %s : delegates room operations\n", m.Service.Name, m.Registry.Name))
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

func generateMarkdown(m *ComponentModel) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("# %s — детализация кода L4\n\n", m.Config.Title))
	buf.WriteString(fmt.Sprintf("Этот раздел описывает внутреннее устройство, классы, доменные сущности, контракты и структуры компонента **%s**, входящего в контейнер `%s`.\n\n", m.Config.Name, m.Config.Container))
	buf.WriteString(fmt.Sprintf("> %s\n\n", m.Config.Description))

	buf.WriteString("```plantuml\n")
	buf.WriteString("@startuml\n")
	buf.WriteString(fmt.Sprintf("!include architecture/c4-plant-uml/plantuml/code/backend/%s.puml\n", m.Config.Slug))
	buf.WriteString("@enduml\n")
	buf.WriteString("```\n\n")

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
			if isExported(method.Name) {
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

	buf.WriteString("## 3. Сценарии использования (Application Layer)\n\n")
	buf.WriteString(fmt.Sprintf("Прикладной сервис `%s` оркестрирует выполнение бизнес-сценариев:\n\n", m.Service.Name))
	for _, method := range m.Service.Methods {
		buf.WriteString(fmt.Sprintf("* `+%s(%s): %s`\n", method.Name, method.Params, method.ReturnType))
	}
	buf.WriteString("\n")

	if m.Registry != nil {
		buf.WriteString(fmt.Sprintf("### Стратегии прикладного уровня (%s)\n\n", m.Registry.Name))
		buf.WriteString(fmt.Sprintf("Для оркестрации провижининга комнат используется объект стратегии `%s` с реестром реализаций `map[MeetingFormat]MeetingRoomStrategy`.\n\n", m.Registry.Name))
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

func formatType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "int", "int64":
			return "int"
		case "string":
			return "string"
		case "bool":
			return "bool"
		default:
			return t.Name
		}
	case *ast.SelectorExpr:
		pkg := formatType(t.X)
		if pkg == "uuid" && t.Sel.Name == "UUID" {
			return "UUID"
		}
		if pkg == "time" && t.Sel.Name == "Time" {
			return "Timestamp"
		}
		if pkg == "context" && t.Sel.Name == "Context" {
			return "Context"
		}
		return t.Sel.Name
	case *ast.StarExpr:
		return formatType(t.X)
	case *ast.ArrayType:
		return fmt.Sprintf("List<%s>", formatType(t.Elt))
	case *ast.MapType:
		return fmt.Sprintf("Map<%s, %s>", formatType(t.Key), formatType(t.Value))
	default:
		return "any"
	}
}

func formatParams(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 {
		return ""
	}
	var res []string
	for _, f := range fields.List {
		typeName := formatType(f.Type)
		if len(f.Names) == 0 {
			res = append(res, typeName)
		} else {
			for _, name := range f.Names {
				res = append(res, fmt.Sprintf("%s: %s", name.Name, typeName))
			}
		}
	}
	return strings.Join(res, ", ")
}

func formatResults(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 {
		return "void"
	}
	var res []string
	for _, f := range fields.List {
		res = append(res, formatType(f.Type))
	}
	if len(res) == 1 {
		return res[0]
	}
	// Filter error
	var filtered []string
	hasError := false
	for _, r := range res {
		if r == "error" {
			hasError = true
		} else {
			filtered = append(filtered, r)
		}
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	if len(filtered) == 0 && hasError {
		return "error"
	}
	return fmt.Sprintf("(%s)", strings.Join(res, ", "))
}

func cleanRecvType(s string) string {
	s = strings.TrimPrefix(s, "*")
	return s
}

func isExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	first := name[0]
	return first >= 'A' && first <= 'Z'
}

func isAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func containsTypeName(s, typeName string) bool {
	idx := strings.Index(s, typeName)
	for idx != -1 {
		beforeOk := idx == 0 || !isAlphaNum(s[idx-1])
		endIdx := idx + len(typeName)
		afterOk := endIdx == len(s) || !isAlphaNum(s[endIdx])
		if beforeOk && afterOk {
			return true
		}
		nextIdx := strings.Index(s[idx+1:], typeName)
		if nextIdx == -1 {
			break
		}
		idx = idx + 1 + nextIdx
	}
	return false
}
