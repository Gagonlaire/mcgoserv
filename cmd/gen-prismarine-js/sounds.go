package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/template"
)

type SoundDefinition struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GenSound struct {
	PascalName string
	ID         int
	Name       string
}

type SoundTemplateData struct {
	Sounds     []GenSound
	MaxSoundID int
}

func generateSounds(rawSounds io.ReadCloser, data map[string]any) error {
	const (
		outputFile = "internal/mc/sound/sound_gen.go"
		tmplFile   = "cmd/gen-prismarine-js/tmpl/sounds.tmpl"
	)

	var soundDefs []SoundDefinition
	if err := json.NewDecoder(rawSounds).Decode(&soundDefs); err != nil {
		return err
	}

	maxID := 0
	processed := make([]GenSound, len(soundDefs))
	for i, s := range soundDefs {
		if s.ID > maxID {
			maxID = s.ID
		}
		processed[i] = GenSound{
			PascalName: toPascalCase(s.Name),
			ID:         s.ID,
			Name:       s.Name,
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

	if err := tmpl.Execute(outFile, SoundTemplateData{Sounds: processed, MaxSoundID: maxID}); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	data["sounds"] = soundDefs
	return nil
}
