package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"text/template"
)

type MaterialDefinition map[string]map[string]float32

type MaterialTemplateData map[string]map[int]float32

func generateMaterials(rawMaterials io.ReadCloser, data map[string]any) error {
	const (
		outputFile = "internal/mcdata/materials_gen.go"
		tmplFile   = "cmd/gen-prismarine-js/tmpl/materials.tmpl"
	)

	var materials MaterialDefinition
	if err := json.NewDecoder(rawMaterials).Decode(&materials); err != nil {
		return err
	}

	var processedMaterials MaterialTemplateData = make(map[string]map[int]float32)
	for matName, tools := range materials {
		processedMaterials[matName] = make(map[int]float32)

		for toolIDStr, multiplier := range tools {
			toolID, err := strconv.Atoi(toolIDStr)
			if err != nil {
				continue
			}
			processedMaterials[matName][toolID] = multiplier
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

	if err := tmpl.Execute(outFile, processedMaterials); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	data["materials"] = processedMaterials

	return nil
}
