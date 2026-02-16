package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"unicode"
)

const (
	basePrismaJSURl    = "https://raw.githubusercontent.com/PrismarineJS/minecraft-data/master/data/"
	projectVersionFile = "version.json"
)

type DataPaths struct {
	Pc      map[string]map[string]string `json:"pc"`
	Bedrock map[string]map[string]string `json:"bedrock"`
}

type VersionData struct {
	ProtocolVersion int    `json:"protocol_version"`
	Version         string `json:"game_version"`
}

func main() {
	resp, err := http.Get(basePrismaJSURl + "dataPaths.json")
	if err != nil || resp.StatusCode != http.StatusOK {
		panic("Cannot fetch " + basePrismaJSURl)
	}

	var dataPaths DataPaths
	if err := json.NewDecoder(resp.Body).Decode(&dataPaths); err != nil {
		panic("Cannot decode " + basePrismaJSURl)
	}
	_ = resp.Body.Close()

	fileContent, err := os.ReadFile(projectVersionFile)
	if err != nil {
		panic(err)
	}

	var versionData VersionData
	if err := json.Unmarshal(fileContent, &versionData); err != nil {
		panic(err)
	}

	versionInfo, ok := dataPaths.Pc[versionData.Version]
	if !ok {
		panic("Version not found: " + versionData.Version)
	}

	blocksURL := fmt.Sprintf("%s%s/blocks.json", basePrismaJSURl, versionInfo["blocks"])
	respBlocks, err := http.Get(blocksURL)
	if err != nil || respBlocks.StatusCode != http.StatusOK {
		panic("Cannot fetch " + blocksURL)
	}

	blockIDMap, err := generateBlocks(respBlocks.Body)
	if err != nil {
		panic(err)
	}
	_ = respBlocks.Body.Close()

	itemsURL := fmt.Sprintf("%s%s/items.json", basePrismaJSURl, versionInfo["items"])
	respItems, err := http.Get(itemsURL)
	if err != nil || respItems.StatusCode != http.StatusOK {
		panic("Cannot fetch " + itemsURL)
	}

	if err := generateItems(respItems.Body, blockIDMap); err != nil {
		panic(err)
	}
	_ = respItems.Body.Close()
}

func toPascalCase(input string) string {
	if idx := strings.Index(input, ":"); idx != -1 {
		input = input[idx+1:]
	}

	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == '_' || r == ' ' || r == '-'
	})

	for i, part := range parts {
		if len(part) > 0 {
			r := []rune(part)
			r[0] = unicode.ToUpper(r[0])
			parts[i] = string(r)
		}
	}

	name := strings.Join(parts, "")
	if name == "Map" {
		return "MapBlock"
	}
	if name == "Func" {
		return "FuncBlock"
	}

	return name
}
