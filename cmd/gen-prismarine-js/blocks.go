package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
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
	Resistance     float32
	StackSize      int
	Diggable       bool
	Material       string
	Transparent    bool
	EmitLight      int
	FilterLight    int
	BoundingBox    string
	Drops          []int
	HarvestTools   map[string]bool
	MinStateID     int
	MaxStateID     int
	DefaultStateID int
	States         []GenProperty
	Sounds         map[string]int
}

type BlockTemplateData struct {
	Blocks     []GenBlock
	MaxBlockID int
	MaxStateID int
}

func generateBlocks(rawBlockDefinitions io.ReadCloser, data map[string]any) error {
	const (
		outputFile = "internal/mcdata/blocks_gen.go"
		tmplFile   = "cmd/gen-prismarine-js/tmpl/blocks.tmpl"
	)

	var blockDefinitions []BlockDefinition
	if err := json.NewDecoder(rawBlockDefinitions).Decode(&blockDefinitions); err != nil {
		return err
	}

	allSounds := data["sounds"].([]SoundDefinition)
	soundLookup := make(map[string]int)
	for _, s := range allSounds {
		soundLookup[s.Name] = s.ID
	}

	blockIDMap := make(map[string]int)
	var processedBlocks []GenBlock

	maxBlockID := 0
	maxStateID := 0

	for _, b := range blockDefinitions {
		blockIDMap[b.Name] = b.ID

		// states
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
			Resistance:     b.Resistance,
			StackSize:      b.StackSize,
			Diggable:       b.Diggable,
			Material:       b.Material,
			Transparent:    b.Transparent,
			EmitLight:      b.EmitLight,
			FilterLight:    b.FilterLight,
			BoundingBox:    b.BoundingBox,
			Drops:          b.Drops,
			HarvestTools:   b.HarvestTools,
			MinStateID:     b.MinStateID,
			MaxStateID:     b.MaxStateID,
			DefaultStateID: b.Default,
			States:         genStates,
		})
	}

	for _, sound := range allSounds {
		if strings.HasPrefix(sound.Name, "block.") {
			parts := strings.SplitN(sound.Name, ".", 3)
			if len(parts) == 3 {
				blockName := parts[1]
				if blockID, ok := blockIDMap[blockName]; ok {
					if _, exists := processedBlocks[blockID].Sounds[sound.Name]; !exists {
						if processedBlocks[blockID].Sounds == nil {
							processedBlocks[blockID].Sounds = make(map[string]int)
						}

						processedBlocks[blockID].Sounds[parts[2]] = sound.ID
					}
				}
			}
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

	tmplData := BlockTemplateData{
		Blocks:     processedBlocks,
		MaxBlockID: maxBlockID,
		MaxStateID: maxStateID,
	}

	if err := tmpl.Execute(outFile, tmplData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	data["blocks"] = blockIDMap

	return nil
}
