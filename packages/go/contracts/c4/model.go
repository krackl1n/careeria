package c4

import (
	"fmt"
	"go/ast"
	"strings"
)

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

type Layer struct {
	Name        string
	Description string
	Classes     []StructInfo
	Interfaces  []InterfaceInfo
	Enums       []EnumInfo
}

type Relation struct {
	From  string
	To    string
	Arrow string // "-->", "..>", "*--", "o--", "<|.."
	Label string
}

type LayoutRule struct {
	From      string
	To        string
	Direction string // "down" or "right"
}

type ComponentModel struct {
	Config             ComponentConfig
	Kind               ComponentKind
	Layers             []Layer
	Relations          []Relation
	LayoutRules        []LayoutRule
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

func FormatType(expr ast.Expr) string {
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
		pkg := FormatType(t.X)
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
		return FormatType(t.X)
	case *ast.ArrayType:
		return fmt.Sprintf("List<%s>", FormatType(t.Elt))
	case *ast.MapType:
		return fmt.Sprintf("Map<%s, %s>", FormatType(t.Key), FormatType(t.Value))
	case *ast.InterfaceType:
		return "any"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func FormatParams(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 {
		return ""
	}
	var res []string
	for _, f := range fields.List {
		typ := FormatType(f.Type)
		if len(f.Names) == 0 {
			res = append(res, typ)
		} else {
			for _, name := range f.Names {
				res = append(res, fmt.Sprintf("%s: %s", name.Name, typ))
			}
		}
	}
	return strings.Join(res, ", ")
}

func FormatResults(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 {
		return "void"
	}
	var res []string
	for _, f := range fields.List {
		res = append(res, FormatType(f.Type))
	}
	if len(res) == 1 {
		return res[0]
	}
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

func CleanRecvType(s string) string {
	return strings.TrimPrefix(s, "*")
}

func IsExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	first := name[0]
	return first >= 'A' && first <= 'Z'
}

func IsAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func ContainsTypeName(s, typeName string) bool {
	idx := strings.Index(s, typeName)
	for idx != -1 {
		beforeOk := idx == 0 || !IsAlphaNum(s[idx-1])
		endIdx := idx + len(typeName)
		afterOk := endIdx == len(s) || !IsAlphaNum(s[endIdx])
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

