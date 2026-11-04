package c4

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
)

// AnalyzeComponent routes analysis based on ComponentKind.
func AnalyzeComponent(cfg ComponentConfig) (*ComponentModel, error) {
	switch cfg.Kind {
	case KindTransport:
		return AnalyzeTransportComponent(cfg)
	case KindAdapter:
		return AnalyzeAdapterComponent(cfg)
	case KindGeneric:
		return AnalyzeGenericComponent(cfg)
	default:
		if len(cfg.Layers) > 0 {
			return AnalyzeGenericComponent(cfg)
		}
		return AnalyzeDomainComponent(cfg)
	}
}

// AnalyzeDomainComponent analyzes Go DDD code (Application Service, Domain Aggregates, Policies, Ports).
func AnalyzeDomainComponent(cfg ComponentConfig) (*ComponentModel, error) {
	model := &ComponentModel{
		Config: cfg,
		Kind:   KindDomain,
	}

	fset := token.NewFileSet()

	// 1. Parse Usecase Package
	usecasePkgs, err := parser.ParseDir(fset, cfg.UsecasePkg, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse usecase pkg %s: %w", cfg.UsecasePkg, err)
	}

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
					typeName := typeSpec.Name.Name

					switch t := typeSpec.Type.(type) {
					case *ast.StructType:
						if typeName == "Service" || typeName == cfg.ServiceName || strings.HasSuffix(typeName, "Service") {
							if model.Service.Name == "" {
								displayName := cfg.ServiceName
								if displayName == "" {
									displayName = typeName
								}
								model.Service.Name = displayName
								model.Service.Role = "Application Service"
								for _, field := range t.Fields.List {
									fieldType := FormatType(field.Type)
									if len(field.Names) == 0 {
										model.Service.Fields = append(model.Service.Fields, FieldInfo{
											Name: fieldType,
											Type: "",
										})
									} else {
										for _, name := range field.Names {
											model.Service.Fields = append(model.Service.Fields, FieldInfo{
												Name: name.Name,
												Type: fieldType,
											})
										}
									}
								}
							}
						}

					case *ast.InterfaceType:
						iface := InterfaceInfo{
							Name: typeName,
						}
						for _, method := range t.Methods.List {
							if funcType, ok := method.Type.(*ast.FuncType); ok {
								for _, name := range method.Names {
									iface.Methods = append(iface.Methods, MethodInfo{
										Name:       name.Name,
										Params:     FormatParams(funcType.Params),
										ReturnType: FormatResults(funcType.Results),
									})
								}
							}
						}
						// Only treat as external contract if it represents a port/repository
						if strings.HasSuffix(typeName, "Repository") ||
							strings.HasSuffix(typeName, "Checker") ||
							strings.HasSuffix(typeName, "Port") ||
							strings.HasSuffix(typeName, "Client") ||
							strings.HasSuffix(typeName, "Reader") ||
							strings.HasSuffix(typeName, "Writer") ||
							strings.HasSuffix(typeName, "Gateway") ||
							strings.HasSuffix(typeName, "Publisher") {
							model.Ports = append(model.Ports, iface)
						}
					}
				}
			}

			// Service methods
			for _, decl := range file.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
					continue
				}
				recvType := CleanRecvType(FormatType(funcDecl.Recv.List[0].Type))
				if (recvType == "Service" || recvType == model.Service.Name) && funcDecl.Name.IsExported() {
					model.Service.Methods = append(model.Service.Methods, MethodInfo{
						Name:       funcDecl.Name.Name,
						Params:     FormatParams(funcDecl.Type.Params),
						ReturnType: FormatResults(funcDecl.Type.Results),
					})
				}
			}
		}
	}

	sort.Slice(model.Service.Methods, func(i, j int) bool {
		return model.Service.Methods[i].Name < model.Service.Methods[j].Name
	})

	// 2. Parse Domain Package
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
					lastType := ""
					for _, spec := range genDecl.Specs {
						valueSpec, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}
						if valueSpec.Type != nil {
							lastType = FormatType(valueSpec.Type)
						}
						typeName := lastType
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
							}{Name: name.Name, Type: typeName, Val: val})
						}
					}
				}

				if genDecl.Tok == token.TYPE {
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
								fieldType := FormatType(field.Type)
								if len(field.Names) == 0 {
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
								allDomainEnums[typeName] = &EnumInfo{Name: typeName}
							}
						}
					}
				}
			}
		}
	}

	// Methods on domain structs
	for _, pkg := range domainPkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
					continue
				}
				recvType := CleanRecvType(FormatType(funcDecl.Recv.List[0].Type))
				if sInfo, ok := allDomainStructs[recvType]; ok {
					sInfo.Methods = append(sInfo.Methods, MethodInfo{
						Name:       funcDecl.Name.Name,
						Params:     FormatParams(funcDecl.Type.Params),
						ReturnType: FormatResults(funcDecl.Type.Results),
					})
				}
			}
		}
	}

	// Populate enum values generically from collected constants
	for _, c := range allConstants {
		if c.Type != "" {
			if enum, ok := allDomainEnums[c.Type]; ok {
				enum.Values = append(enum.Values, c.Val)
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
						recvType := CleanRecvType(FormatType(funcDecl.Recv.List[0].Type))
						for i := range model.Policies {
							if model.Policies[i].Name == recvType {
								model.Policies[i].Methods = append(model.Policies[i].Methods, MethodInfo{
									Name:       funcDecl.Name.Name,
									Params:     FormatParams(funcDecl.Type.Params),
									ReturnType: FormatResults(funcDecl.Type.Results),
								})
							}
						}
					}
				}
			}
		}
	}

	// 4. Parse Subpackages (e.g. Strategy patterns)
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
													Params:     FormatParams(funcType.Params),
													ReturnType: FormatResults(funcType.Results),
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
													Params:     FormatParams(funcType.Params),
													ReturnType: FormatResults(funcType.Results),
												})
											}
										}
									}
									model.StrategyIface = &iface
								}

							case *ast.StructType:
								if strings.HasSuffix(typeSpec.Name.Name, "Request") || strings.HasSuffix(typeSpec.Name.Name, "Result") || !IsExported(typeSpec.Name.Name) {
									continue
								}

								// Generic Strategy Registry detection: has map to a Strategy interface or name contains Registry / Strategy Coordinator
								isStrategyRegistry := false
								var registryFields []FieldInfo
								for _, field := range t.Fields.List {
									fieldType := FormatType(field.Type)
									if _, ok := field.Type.(*ast.MapType); ok && strings.Contains(fieldType, "Strategy") {
										isStrategyRegistry = true
									}
									for _, name := range field.Names {
										registryFields = append(registryFields, FieldInfo{
											Name: name.Name,
											Type: fieldType,
										})
									}
								}

								if isStrategyRegistry || strings.HasSuffix(typeSpec.Name.Name, "Registry") {
									reg := StructInfo{
										Name:   typeSpec.Name.Name,
										Role:   "Strategy",
										Fields: registryFields,
									}
									model.Registry = &reg
								} else if strings.HasSuffix(typeSpec.Name.Name, "Strategy") {
									st := StructInfo{
										Name: typeSpec.Name.Name,
										Role: "Strategy",
									}
									for _, field := range t.Fields.List {
										fieldType := FormatType(field.Type)
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
						recvType := CleanRecvType(FormatType(funcDecl.Recv.List[0].Type))
						if model.Registry != nil && model.Registry.Name == recvType && funcDecl.Name.IsExported() {
							model.Registry.Methods = append(model.Registry.Methods, MethodInfo{
								Name:       funcDecl.Name.Name,
								Params:     FormatParams(funcDecl.Type.Params),
								ReturnType: FormatResults(funcDecl.Type.Results),
							})
						}
						for i := range model.StrategyImpls {
							if model.StrategyImpls[i].Name == recvType && funcDecl.Name.IsExported() {
								model.StrategyImpls[i].Methods = append(model.StrategyImpls[i].Methods, MethodInfo{
									Name:       funcDecl.Name.Name,
									Params:     FormatParams(funcDecl.Type.Params),
									ReturnType: FormatResults(funcDecl.Type.Results),
								})
							}
						}
					}
				}
			}
		}
	}

	// 5. Parse Cross-Domain Package (referenced external aggregates or projections)
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
										if ContainsTypeName(m.ReturnType, typeName) || ContainsTypeName(m.Params, typeName) {
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
											if ContainsTypeName(m.Params, typeName) {
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
										fieldType := FormatType(field.Type)
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

	// Deterministic sorting
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
			return !cI // Implementations using clients go on the right edge
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
