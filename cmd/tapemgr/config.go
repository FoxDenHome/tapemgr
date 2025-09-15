package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	LoaderDevice string   `json:"loader-device"`
	DriveDevice  string   `json:"drive-device"`
	TapeMount    string   `json:"tape-mount"`
	TapeFileKey  string   `json:"tape-file-key"`
	TapePathKey  string   `json:"tape-path-key"`
	TapesPath    string   `json:"tapes-path"`
	DryRun       bool     `json:"dry-run"`
	Targets      []string `json:"targets"`
}

func loadConfig(path string) (Config, error) {
	reader, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer func() {
		_ = reader.Close()
	}()

	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()

	var config Config
	err = decoder.Decode(&config)
	if err != nil {
		return Config{}, err
	}
	return config, nil
}
