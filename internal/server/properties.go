package server

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Properties struct {
	MaxPlayers          int    `property:"max-players" default:"20"`
	Motd                string `property:"motd" default:"A Minecraft Server"`
	ViewDistance        int    `property:"view-distance" default:"10"`
	SimulationDistance  int    `property:"simulation-distance" default:"10"`
	Hardcore            bool   `property:"hardcore" default:"false"`
	EnableRespawnScreen bool   `property:"enable-respawn-screen" default:"true"`
	LevelName           string `property:"level-name" default:"world"`
	ServerIp            string `property:"server-ip" default:""`
	ServerPort          int    `property:"server-port" default:"25565"`
}

func LoadProperties(path string) (*Properties, error) {
	defaults := getTypeDefaults()
	props := &Properties{}
	fileProps := make(map[string]string)

	if _, err := os.Stat(path); err == nil {
		var err error
		fileProps, err = readPropertiesFile(path)
		if err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	modified := false
	for k, v := range defaults {
		if _, exists := fileProps[k]; !exists {
			fileProps[k] = v
			modified = true
		}
	}

	if modified || len(fileProps) == 0 {
		if err := writeProperties(path, fileProps); err != nil {
			return nil, err
		}
	}
	if err := populateStruct(props, fileProps); err != nil {
		return nil, err
	}

	return props, nil
}

func getTypeDefaults() map[string]string {
	defaults := make(map[string]string)
	t := reflect.TypeOf(Properties{})
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		key := field.Tag.Get("property")
		def := field.Tag.Get("default")
		if key != "" {
			defaults[key] = def
		}
	}

	return defaults
}

func readPropertiesFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	props := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			props[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	return props, scanner.Err()
}

func writeProperties(path string, props map[string]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Fprintln(writer, "#Minecraft server properties")
	fmt.Fprintf(writer, "#%s\n", time.Now().Format(time.RFC1123))

	for _, k := range keys {
		fmt.Fprintf(writer, "%s=%s\n", k, props[k])
	}

	return writer.Flush()
}

func populateStruct(props *Properties, data map[string]string) error {
	v := reflect.ValueOf(props).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		key := field.Tag.Get("property")
		if key == "" {
			continue
		}

		valStr, ok := data[key]
		if !ok {
			valStr = field.Tag.Get("default")
		}

		fieldVal := v.Field(i)
		switch fieldVal.Kind() {
		case reflect.String:
			fieldVal.SetString(valStr)
		case reflect.Int:
			intVal, err := strconv.Atoi(valStr)
			if err != nil {
				return fmt.Errorf("invalid int value for %s: %s", key, valStr)
			}
			fieldVal.SetInt(int64(intVal))
		case reflect.Bool:
			boolVal, err := strconv.ParseBool(valStr)
			if err != nil {
				return fmt.Errorf("invalid bool value for %s: %s", key, valStr)
			}
			fieldVal.SetBool(boolVal)
		default:
			panic("Missing case for field type: " + fieldVal.Kind().String())
		}
	}

	return nil
}
