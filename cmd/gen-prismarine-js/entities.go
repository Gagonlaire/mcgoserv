package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/template"
)

type EntityDefinition struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	DisplayName string  `json:"displayName"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	Type        string  `json:"type"`
	Category    string  `json:"category"`
}

type GenEntity struct {
	PascalName  string
	ID          int
	Name        string
	DisplayName string
	Width       float64
	Height      float64
	Type        string
	Category    string
}

func generateEntities(rawEntities io.ReadCloser, _ map[string]any) error {
	const (
		outputFile = "internal/mc/entity/entity_gen.go"
		tmplFile   = "cmd/gen-prismarine-js/tmpl/entities.tmpl"
	)

	var definitions []EntityDefinition
	if err := json.NewDecoder(rawEntities).Decode(&definitions); err != nil {
		return err
	}

	maxID := 0
	for _, def := range definitions {
		if def.ID > maxID {
			maxID = def.ID
		}
	}

	processedEntities := make([]*GenEntity, maxID+1)
	for _, def := range definitions {
		pascalName := toPascalCase(def.Name)
		if pascalName == "" {
			continue
		}
		processedEntities[def.ID] = &GenEntity{
			PascalName:  pascalName + "ID",
			ID:          def.ID,
			Name:        def.Name,
			DisplayName: def.DisplayName,
			Width:       def.Width,
			Height:      def.Height,
			Type:        def.Type,
			Category:    def.Category,
		}
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

	if err := tmpl.Execute(outFile, processedEntities); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
