package openapi

import "embed"

//go:embed careeria.swagger.json
var files embed.FS

func Specification() []byte {
	content, err := files.ReadFile("careeria.swagger.json")
	if err != nil {
		panic(err)
	}

	return content
}

