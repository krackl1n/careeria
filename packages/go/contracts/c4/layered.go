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

// ParsePackageToLayer parses a Go package directory and converts its types into a Layer.
func ParsePackageToLayer(pkgPath, layerName, defaultRole string) (*Layer, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkgPath, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse dir %s: %w", pkgPath, err)
	}

	layer := &Layer{
		Name: layerName,
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			// Extract structs and interfaces
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || !typeSpec.Name.IsExported() {
						continue
					}
					typeName := typeSpec.Name.Name
					switch t := typeSpec.Type.(type) {
					case *ast.StructType:
						st := StructInfo{
							Name: typeName,
							Role: InferRole(typeName, defaultRole),
						}
						if t.Fields != nil {
							for _, field := range t.Fields.List {
								fieldType := FormatType(field.Type)
								if len(field.Names) == 0 {
									st.Fields = append(st.Fields, FieldInfo{Name: fieldType, Type: ""})
								} else {
									for _, name := range field.Names {
										st.Fields = append(st.Fields, FieldInfo{Name: name.Name, Type: fieldType})
									}
								}
							}
						}
						layer.Classes = append(layer.Classes, st)

					case *ast.InterfaceType:
						iface := InterfaceInfo{
							Name: typeName,
						}
						if t.Methods != nil {
							for _, m := range t.Methods.List {
								if ft, ok := m.Type.(*ast.FuncType); ok {
									for _, name := range m.Names {
										iface.Methods = append(iface.Methods, MethodInfo{
											Name:       name.Name,
											Params:     FormatParams(ft.Params),
											ReturnType: FormatResults(ft.Results),
										})
									}
								}
							}
						}
						layer.Interfaces = append(layer.Interfaces, iface)
					}
				}
			}

			// Extract methods for structs
			for _, decl := range file.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
					continue
				}
				recvType := CleanRecvType(FormatType(funcDecl.Recv.List[0].Type))
				for i := range layer.Classes {
					if layer.Classes[i].Name == recvType && funcDecl.Name.IsExported() {
						layer.Classes[i].Methods = append(layer.Classes[i].Methods, MethodInfo{
							Name:       funcDecl.Name.Name,
							Params:     FormatParams(funcDecl.Type.Params),
							ReturnType: FormatResults(funcDecl.Type.Results),
						})
					}
				}
			}
		}
	}

	// Sort elements
	sort.Slice(layer.Classes, func(i, j int) bool { return layer.Classes[i].Name < layer.Classes[j].Name })
	sort.Slice(layer.Interfaces, func(i, j int) bool { return layer.Interfaces[i].Name < layer.Interfaces[j].Name })

	return layer, nil
}

// InferRole assigns an architectural role / stereotype based on naming conventions.
func InferRole(name, defaultRole string) string {
	if defaultRole != "" {
		return defaultRole
	}
	switch {
	case strings.HasSuffix(name, "Server"):
		return "gRPC Server"
	case strings.HasSuffix(name, "Handler"):
		return "Handler"
	case strings.HasSuffix(name, "Interceptor") || strings.HasSuffix(name, "Middleware"):
		return "Middleware"
	case strings.HasSuffix(name, "Repository"):
		return "Repository Adapter"
	case strings.HasSuffix(name, "Client"):
		return "Client Adapter"
	case strings.HasSuffix(name, "Publisher"):
		return "Publisher Adapter"
	case strings.HasSuffix(name, "Consumer"):
		return "Event Consumer"
	case strings.HasSuffix(name, "Worker"):
		return "Background Worker"
	case strings.HasSuffix(name, "Pool") || strings.HasSuffix(name, "DB"):
		return "Database Pool"
	default:
		return "Component"
	}
}

// AnalyzeTransportComponent builds an L4 model for transport layers (servers, handlers, interceptors).
func AnalyzeTransportComponent(cfg ComponentConfig) (*ComponentModel, error) {
	model := &ComponentModel{
		Config: cfg,
		Kind:   KindTransport,
	}

	// 1. Inbound Transport Layer (Handlers / Servers)
	if cfg.TransportPkg != "" {
		layer, err := ParsePackageToLayer(cfg.TransportPkg, "Inbound Transport (gRPC / HTTP)", "")
		if err != nil {
			return nil, fmt.Errorf("parse transport pkg: %w", err)
		}
		model.Layers = append(model.Layers, *layer)
	}

	// 2. Middlewares & Interceptors
	if cfg.InterceptorPkg != "" {
		layer, err := ParsePackageToLayer(cfg.InterceptorPkg, "Middlewares & Interceptors", "Middleware")
		if err != nil {
			return nil, fmt.Errorf("parse interceptor pkg: %w", err)
		}
		model.Layers = append(model.Layers, *layer)
	}

	// 3. Port / Application Contracts
	if cfg.PortPkg != "" {
		layer, err := ParsePackageToLayer(cfg.PortPkg, "Invoked UseCase Contracts", "UseCase Contract")
		if err != nil {
			return nil, fmt.Errorf("parse port pkg: %w", err)
		}
		model.Layers = append(model.Layers, *layer)
	}

	InferRelationsAndLayout(model)
	return model, nil
}

// AnalyzeAdapterComponent builds an L4 model for adapters & infrastructure.
func AnalyzeAdapterComponent(cfg ComponentConfig) (*ComponentModel, error) {
	model := &ComponentModel{
		Config: cfg,
		Kind:   KindAdapter,
	}

	// 1. Adapter Implementations (Repositories, Clients, Outbox)
	if cfg.AdapterPkg != "" {
		layer, err := ParsePackageToLayer(cfg.AdapterPkg, "Adapters & Implementations", "")
		if err != nil {
			return nil, fmt.Errorf("parse adapter pkg: %w", err)
		}
		model.Layers = append(model.Layers, *layer)
	}

	// 2. Implemented Ports / Contracts
	if cfg.PortPkg != "" {
		layer, err := ParsePackageToLayer(cfg.PortPkg, "Domain Ports (Implemented)", "Domain Port")
		if err != nil {
			return nil, fmt.Errorf("parse port pkg: %w", err)
		}
		model.Layers = append(model.Layers, *layer)
	}

	// 3. Infrastructure & Database Pools
	if cfg.InfraPkg != "" {
		layer, err := ParsePackageToLayer(cfg.InfraPkg, "Infrastructure & Drivers", "Infrastructure Driver")
		if err != nil {
			return nil, fmt.Errorf("parse infra pkg: %w", err)
		}
		model.Layers = append(model.Layers, *layer)
	}

	InferRelationsAndLayout(model)
	return model, nil
}

// AnalyzeGenericComponent builds an L4 model from arbitrary configured layers.
func AnalyzeGenericComponent(cfg ComponentConfig) (*ComponentModel, error) {
	model := &ComponentModel{
		Config: cfg,
		Kind:   KindGeneric,
	}

	for _, lcfg := range cfg.Layers {
		mergedLayer := Layer{
			Name: lcfg.Name,
		}
		for _, pkg := range lcfg.Packages {
			subLayer, err := ParsePackageToLayer(pkg, lcfg.Name, lcfg.Role)
			if err != nil {
				return nil, fmt.Errorf("parse layer %s pkg %s: %w", lcfg.Name, pkg, err)
			}
			mergedLayer.Classes = append(mergedLayer.Classes, subLayer.Classes...)
			mergedLayer.Interfaces = append(mergedLayer.Interfaces, subLayer.Interfaces...)
			mergedLayer.Enums = append(mergedLayer.Enums, subLayer.Enums...)
		}
		model.Layers = append(model.Layers, mergedLayer)
	}

	InferRelationsAndLayout(model)
	return model, nil
}

// InferRelationsAndLayout automatically connects types across layers and establishes vertical layer flow.
func InferRelationsAndLayout(model *ComponentModel) {
	allTypes := make(map[string]bool)
	for _, layer := range model.Layers {
		for _, c := range layer.Classes {
			allTypes[c.Name] = true
		}
		for _, iface := range layer.Interfaces {
			allTypes[iface.Name] = true
		}
	}

	seenRelations := make(map[string]bool)
	addRel := func(from, to, arrow, label string) {
		key := fmt.Sprintf("%s|%s|%s", from, to, arrow)
		if !seenRelations[key] && from != to {
			seenRelations[key] = true
			model.Relations = append(model.Relations, Relation{
				From:  from,
				To:    to,
				Arrow: arrow,
				Label: label,
			})
		}
	}

	for _, layer := range model.Layers {
		for _, c := range layer.Classes {
			for _, f := range c.Fields {
				for target := range allTypes {
					if ContainsTypeName(f.Type, target) || f.Name == target {
						addRel(c.Name, target, "-->", "uses")
					}
				}
			}
			for _, m := range c.Methods {
				for target := range allTypes {
					if ContainsTypeName(m.Params, target) || ContainsTypeName(m.ReturnType, target) {
						addRel(c.Name, target, "..>", "uses")
					}
				}
			}
		}
	}

	// Layout rules between layers
	for i := 0; i < len(model.Layers)-1; i++ {
		firstTop := getFirstElement(model.Layers[i])
		firstBottom := getFirstElement(model.Layers[i+1])
		if firstTop != "" && firstBottom != "" {
			model.LayoutRules = append(model.LayoutRules, LayoutRule{
				From:      firstTop,
				To:        firstBottom,
				Direction: "down",
			})
		}
	}
}

func getFirstElement(l Layer) string {
	if len(l.Classes) > 0 {
		return l.Classes[0].Name
	}
	if len(l.Interfaces) > 0 {
		return l.Interfaces[0].Name
	}
	if len(l.Enums) > 0 {
		return l.Enums[0].Name
	}
	return ""
}

