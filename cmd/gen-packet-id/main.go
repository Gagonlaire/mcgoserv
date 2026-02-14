package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

const (
	inputFile  = "internal/mcdata/reports/packets.json"
	outputFile = "internal/packet/packet_id_gen.go"
	tmplFile   = "cmd/gen-packet-id/packet_id.tmpl"
)

type PacketEntry struct {
	ProtocolID int `json:"protocol_id"`
}

type PacketCommon struct {
	Name      string
	CleanName string
	ID        string
}

type DirectionBlock struct {
	Direction string
	Packets   []PacketCommon
}

type StateBlock struct {
	State      string
	Directions []DirectionBlock
}

func main() {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		panic(err)
	}

	var rawMap map[string]map[string]map[string]PacketEntry
	if err := json.Unmarshal(data, &rawMap); err != nil {
		panic(err)
	}

	var stateBlocks []StateBlock

	orderedStates := []string{"handshake", "status", "login", "configuration", "play"}
	orderedDirs := []string{"clientbound", "serverbound"}

	for _, stateName := range orderedStates {
		directions, ok := rawMap[stateName]
		if !ok {
			continue
		}

		var dirBlocks []DirectionBlock

		for _, dirName := range orderedDirs {
			pktsMap, ok := directions[dirName]
			if !ok {
				continue
			}

			var pktNames []string
			for p := range pktsMap {
				pktNames = append(pktNames, p)
			}
			sort.Strings(pktNames)

			var packets []PacketCommon
			for _, pName := range pktNames {
				entry := pktsMap[pName]

				cleanName := toPascalCase(trimNamespace(pName))
				formattedName := toPascalCase(stateName) + toPascalCase(dirName) + cleanName
				hexID := fmt.Sprintf("0x%02X", entry.ProtocolID)

				packets = append(packets, PacketCommon{
					Name:      formattedName,
					CleanName: cleanName,
					ID:        hexID,
				})
			}

			if len(packets) > 0 {
				dirBlocks = append(dirBlocks, DirectionBlock{
					Direction: toPascalCase(dirName),
					Packets:   packets,
				})
			}
		}

		if len(dirBlocks) > 0 {
			stateBlocks = append(stateBlocks, StateBlock{
				State:      toPascalCase(stateName),
				Directions: dirBlocks,
			})
		}
	}

	funcMap := template.FuncMap{}
	tmpl, err := template.New("packet_id.tmpl").Funcs(funcMap).ParseFiles(tmplFile)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(outputFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, stateBlocks); err != nil {
		panic(err)
	}
}

func trimNamespace(s string) string {
	parts := strings.Split(s, ":")
	if len(parts) > 1 {
		return parts[1]
	}
	return s
}

func toPascalCase(s string) string {
	var sb strings.Builder
	nextUpper := true
	for _, r := range s {
		if r == '_' || r == ':' || r == '/' {
			nextUpper = true
			continue
		}
		if nextUpper {
			sb.WriteRune(unicode.ToUpper(r))
			nextUpper = false
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
