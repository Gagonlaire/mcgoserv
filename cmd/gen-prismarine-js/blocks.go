package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/template"
)

type StateProperty struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Values    []string `json:"values"`
	NumValues int      `json:"num_values"`
}

type BlockDefinition struct {
	ID           int             `json:"id"`
	Name         string          `json:"name"`
	DisplayName  string          `json:"displayName"`
	Hardness     float32         `json:"hardness"`
	Resistance   float32         `json:"resistance"`
	StackSize    int             `json:"stackSize"`
	Diggable     bool            `json:"diggable"`
	Material     string          `json:"material"`
	Transparent  bool            `json:"transparent"`
	EmitLight    int             `json:"emitLight"`
	FilterLight  int             `json:"filterLight"`
	MinStateID   int             `json:"minStateId"`
	MaxStateID   int             `json:"maxStateId"`
	Default      int             `json:"defaultState"`
	States       []StateProperty `json:"states"`
	HarvestTools map[string]bool `json:"harvestTools"`
	Drops        []int           `json:"drops"`
	BoundingBox  string          `json:"boundingBox"`
}

type GenProperty struct {
	Name   string
	Values []string
	Stride int
}

type GenBlock struct {
	PascalName     string
	ID             int
	Name           string
	DisplayName    string
	Hardness       float32
	MinStateID     int
	MaxStateID     int
	DefaultStateID int
	States         []GenProperty
}

type BlockTemplateData struct {
	Blocks     []GenBlock
	MaxBlockID int
	MaxStateID int
}

func generateBlocks(rawBlockDefinitions io.ReadCloser) (map[string]int, error) {
	const (
		outputFile = "internal/mc/blocks_gen.go"
		tmplFile   = "cmd/gen-prismarine-js/blocks.tmpl"
	)

	var blockDefinitions []BlockDefinition
	if err := json.NewDecoder(rawBlockDefinitions).Decode(&blockDefinitions); err != nil {
		return nil, err // Return nil map on error
	}

	blockIDMap := make(map[string]int)
	var processedBlocks []GenBlock

	maxBlockID := 0
	maxStateID := 0

	for _, b := range blockDefinitions {
		blockIDMap[b.Name] = b.ID

		pName := toPascalCase(b.Name)
		genStates := make([]GenProperty, len(b.States))
		currentStride := 1

		for i := len(b.States) - 1; i >= 0; i-- {
			prop := b.States[i]
			genStates[i] = GenProperty{
				Name:   prop.Name,
				Values: prop.Values,
				Stride: currentStride,
			}
			currentStride *= len(prop.Values)
		}

		if b.ID > maxBlockID {
			maxBlockID = b.ID
		}
		if b.MaxStateID > maxStateID {
			maxStateID = b.MaxStateID
		}

		processedBlocks = append(processedBlocks, GenBlock{
			PascalName:     pName,
			ID:             b.ID,
			Name:           b.Name,
			DisplayName:    b.DisplayName,
			Hardness:       b.Hardness,
			MinStateID:     b.MinStateID,
			MaxStateID:     b.MaxStateID,
			DefaultStateID: b.Default,
			States:         genStates,
		})
	}

	outFile, err := os.Create(outputFile)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()

	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	data := BlockTemplateData{
		Blocks:     processedBlocks,
		MaxBlockID: maxBlockID,
		MaxStateID: maxStateID,
	}

	if err := tmpl.Execute(outFile, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return blockIDMap, nil
}
