package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/template"
)

type ItemDefinition struct {
	ID                int      `json:"id"`
	Name              string   `json:"name"`
	DisplayName       string   `json:"displayName"`
	StackSize         int      `json:"stackSize"`
	EnchantCategories []string `json:"enchantCategories"`
	RepairWith        []string `json:"repairWith"`
	MaxDurability     int      `json:"maxDurability"`
}

type GenItem struct {
	PascalName        string
	ID                int
	Name              string
	DisplayName       string
	StackSize         int
	BlockID           int
	EnchantCategories []string
	RepairWith        []string
	MaxDurability     int
}

type ItemTemplateData struct {
	Items     []GenItem
	MaxItemID int
}

var itemToBlockAliases = map[string]string{
	"redstone":       "redstone_wire",
	"wheat_seeds":    "wheat",
	"beetroot_seeds": "beetroot",
	"melon_seeds":    "melon_stem",
	"pumpkin_seeds":  "pumpkin_stem",
	"carrot":         "carrots",
	"potato":         "potatoes",
	"sugar_cane":     "reeds",
	"cake":           "cake",
	"string":         "tripwire",
	"sweet_berries":  "sweet_berry_bush",
}

var itemBlockExclusions = map[string]bool{
	"wheat":    true,
	"beetroot": true,
	"melon":    false,
	"stick":    true,
	"fire":     true,
}

func generateItems(rawItemDefinitions io.ReadCloser, data map[string]any) error {
	const (
		outputFile = "internal/mc/items_gen.go"
		tmplFile   = "cmd/gen-prismarine-js/tmpl/items.tmpl"
	)

	var itemDefinitions []ItemDefinition
	if err := json.NewDecoder(rawItemDefinitions).Decode(&itemDefinitions); err != nil {
		return err
	}

	var processedItems []GenItem
	maxItemID := 0
	blockIDs := data["blocks"].(map[string]int)
	for _, item := range itemDefinitions {
		if item.ID > maxItemID {
			maxItemID = item.ID
		}

		bID := -1
		targetBlockName := item.Name

		if alias, ok := itemToBlockAliases[item.Name]; ok {
			targetBlockName = alias
		}
		if itemBlockExclusions[item.Name] {
			targetBlockName = ""
		}
		if targetBlockName != "" {
			if foundID, ok := blockIDs[targetBlockName]; ok {
				bID = foundID
			}
		}

		if item.MaxDurability == 0 {
			// Item is not damageable
			item.MaxDurability = -1
		}
		processedItems = append(processedItems, GenItem{
			PascalName: toPascalCase(item.Name),
			BlockID:    bID,

			ID:                item.ID,
			Name:              item.Name,
			DisplayName:       item.DisplayName,
			StackSize:         item.StackSize,
			EnchantCategories: item.EnchantCategories,
			RepairWith:        item.RepairWith,
			MaxDurability:     item.MaxDurability,
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

	tmplData := ItemTemplateData{
		Items:     processedItems,
		MaxItemID: maxItemID,
	}

	if err := tmpl.Execute(outFile, tmplData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
