package config

import (
	"gopkg.in/yaml.v3"
	"os"
	"path"
)

type Config struct {
	Metrics   Metrics
	Registers Registers
	Actuators Actuators
}

func Read() (*Config, error) {
	configDir := getConfigDir()
	metrics, err := unmarshalFromFile[Metrics](path.Join(configDir, "metrics.yaml"))
	if err != nil {
		return nil, err
	}
	registers, err := unmarshalFromFile[Registers](path.Join(configDir, "registers.yaml"))
	if err != nil {
		return nil, err
	}
	actuators, err := unmarshalFromFile[Actuators](path.Join(configDir, "actuators.yaml"))
	if err != nil {
		return nil, err
	}
	return &Config{*metrics, *registers, *actuators}, nil
}

func getConfigDir() string {
	if koDataPath := os.Getenv("KO_DATA_PATH"); len(koDataPath) > 0 {
		return koDataPath
	}
	return "config"
}

func unmarshalFromFile[T any](filename string) (*T, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var config T
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
