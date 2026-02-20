package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/template"
)

type LanguageEntry struct {
	Identifier string
	Key        string
	Value      string
}

// Ignore client side language keys to reduce binary size
var ignoredPrefixes = []string{
	"accessibility.",
	"addServer.",
	"advMode.",
	"backup.",
	"book.",
	"burnTime.",
	"chat.link.",
	"connect.",
	"controls.",
	"createWorld.",
	"demo.",
	"difficulty.",
	"download.",
	"gui.",
	"joystick.",
	"key.",
	"lanServer.",
	"language.",
	"menu.",
	"merchant.",
	"mount.",
	"narrator.",
	"optimizeWorld.",
	"options.",
	"pack.",
	"realms.",
	"recipies.",
	"resourcePack.",
	"screenshot.",
	"selectServer.",
	"selectWorld.",
	"sign.",
	"soundCategory.",
	"structure_block.",
	"symlink.",
	"title.",
	"tutorial.",
	"wip.",
}

func generateLanguage(rawLanguages io.ReadCloser, _ map[string]any) error {
	const (
		outputFile = "internal/mcdata/language_gen.go"
		tmplFile   = "cmd/gen-prismarine-js/tmpl/language.tmpl"
	)

	var rawMap map[string]string
	if err := json.NewDecoder(rawLanguages).Decode(&rawMap); err != nil {
		return err
	}

	keys := make([]string, 0, len(rawMap))
	for k := range rawMap {
		isIgnored := false
		for _, prefix := range ignoredPrefixes {
			if strings.HasPrefix(k, prefix) {
				isIgnored = true
				break
			}
		}
		if !isIgnored {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var entries []LanguageEntry
	seenIdentifiers := make(map[string]bool)

	for _, k := range keys {
		val := rawMap[k]
		identifier := toPascalCase(k)
		originalID := identifier
		counter := 2

		for seenIdentifiers[identifier] {
			identifier = fmt.Sprintf("%s%d", originalID, counter)
			counter++
		}
		seenIdentifiers[identifier] = true
		entries = append(entries, LanguageEntry{
			Identifier: identifier,
			Key:        k,
			Value:      val,
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

	if err := tmpl.Execute(outFile, entries); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
