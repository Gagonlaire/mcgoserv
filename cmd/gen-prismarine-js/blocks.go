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
	PascalName       string
	ID               int
	Name             string
	DisplayName      string
	Hardness         float32
	Resistance       float32
	StackSize        int
	Diggable         bool
	Material         string
	Transparent      bool
	EmitLight        int
	FilterLight      int
	BoundingBox      string
	Drops            []int
	HarvestTools     map[string]bool
	MinStateID       int
	MaxStateID       int
	DefaultStateID   int
	States           []GenProperty
	SoundGroupPascal string
}

type BlockTemplateData struct {
	Blocks         []GenBlock
	MaxBlockID     int
	MaxStateID     int
	StateIDToBlock []int
}

func generateBlocks(rawBlockDefinitions io.ReadCloser, data map[string]any) error {
	const (
		outputFile = "internal/mc/block/block_gen.go"
		tmplFile   = "cmd/gen-prismarine-js/tmpl/blocks.tmpl"
	)

	var blockDefinitions []BlockDefinition
	if err := json.NewDecoder(rawBlockDefinitions).Decode(&blockDefinitions); err != nil {
		return err
	}

	blockGroupPascal, ok := data["blockGroupPascal"].(map[string]string)
	if !ok {
		return fmt.Errorf("blockGroupPascal missing from generator data; sounds generator must run first")
	}

	blockIDMap := make(map[string]int)
	var processedBlocks []GenBlock

	// Blocks with no sound group are collected rather than reported one at a
	// time: a game version bump usually adds several, and failing on the first
	// would mean one full generate cycle per missing block.
	var ungrouped []string

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

		groupPascal, ok := blockGroupPascal[b.Name]
		if !ok {
			ungrouped = append(ungrouped, b.Name)
			continue
		}

		processedBlocks = append(processedBlocks, GenBlock{
			PascalName:       pName + "ID",
			ID:               b.ID,
			Name:             b.Name,
			DisplayName:      b.DisplayName,
			Hardness:         b.Hardness,
			Resistance:       b.Resistance,
			StackSize:        b.StackSize,
			Diggable:         b.Diggable,
			Material:         b.Material,
			Transparent:      b.Transparent,
			EmitLight:        b.EmitLight,
			FilterLight:      b.FilterLight,
			BoundingBox:      b.BoundingBox,
			Drops:            b.Drops,
			HarvestTools:     b.HarvestTools,
			MinStateID:       b.MinStateID,
			MaxStateID:       b.MaxStateID,
			DefaultStateID:   b.Default,
			States:           genStates,
			SoundGroupPascal: groupPascal,
		})
	}

	if len(ungrouped) > 0 {
		return fmt.Errorf("%d block(s) have no entry in blockToGroup; add them in cmd/gen-prismarine-js/sounds.go:\n\t%s",
			len(ungrouped), strings.Join(ungrouped, "\n\t"))
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

	stateIDToBlock := make([]int, maxStateID+1)
	for _, b := range processedBlocks {
		for s := b.MinStateID; s <= b.MaxStateID; s++ {
			stateIDToBlock[s] = b.ID
		}
	}

	tmplData := BlockTemplateData{
		Blocks:         processedBlocks,
		MaxBlockID:     maxBlockID,
		MaxStateID:     maxStateID,
		StateIDToBlock: stateIDToBlock,
	}

	if err := tmpl.Execute(outFile, tmplData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	data["blocks"] = blockIDMap

	return nil
}
