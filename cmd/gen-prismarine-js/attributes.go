package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
)

type AttributeDefinition struct {
	Name     string  `json:"name"`
	Resource string  `json:"resource"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Default  float64 `json:"default"`
}

type GenAttribute struct {
	PascalName string
	Resource   string
	Min        float64
	Max        float64
	Default    float64
}

func generateAttributes(rawAttributes io.ReadCloser, _ map[string]any) error {
	const (
		outputFile = "internal/mc/attribute/attribute_gen.go"
		tmplFile   = "cmd/gen-prismarine-js/tmpl/attributes.tmpl"
	)

	var definitions []AttributeDefinition
	if err := json.NewDecoder(rawAttributes).Decode(&definitions); err != nil {
		return err
	}

	processed := make([]GenAttribute, 0, len(definitions))
	for _, def := range definitions {
		pascal := toPascalCase(def.Name)
		if pascal == "" {
			continue
		}
		resource := def.Resource
		if idx := strings.IndexByte(resource, ':'); idx != -1 {
			resource = resource[idx+1:]
		}
		processed = append(processed, GenAttribute{
			PascalName: pascal + "ID",
			Resource:   resource,
			Min:        def.Min,
			Max:        def.Max,
			Default:    def.Default,
		})
	}

	outFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	if err := tmpl.Execute(outFile, processed); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
