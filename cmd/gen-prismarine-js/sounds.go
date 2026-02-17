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

type SoundTemplateData struct {
	Sounds     []SoundDefinition
	MaxSoundID int
}

func generateSounds(rawSounds io.ReadCloser, data map[string]any) error {
	const (
		outputFile = "internal/mc/sounds_gen.go"
		tmplFile   = "cmd/gen-prismarine-js/tmpl/sounds.tmpl"
	)

	var sounds []SoundDefinition
	if err := json.NewDecoder(rawSounds).Decode(&sounds); err != nil {
		return err
	}

	maxID := 0
	for _, s := range sounds {
		if s.ID > maxID {
			maxID = s.ID
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

	tmplData := SoundTemplateData{
		Sounds:     sounds,
		MaxSoundID: maxID,
	}

	if err := tmpl.Execute(outFile, tmplData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	data["sounds"] = sounds

	return nil
}
