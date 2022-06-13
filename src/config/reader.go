package config

import (
	"gopkg.in/yaml.v3"
	"os"
	"path"
)

type Config struct {
	Metrics   Metrics
	Registers Registers
}

func Read() (*Config, error) {
	configDir := getConfigDir()
	metricsConfig, err := unmarshalFromFile[Metrics](path.Join(configDir, "metrics.yaml"))
	if err != nil {
		return nil, err
	}
	registersConfig, err := unmarshalFromFile[Registers](path.Join(configDir, "registers.yaml"))
	if err != nil {
		return nil, err
	}
	return &Config{
		Metrics:   *metricsConfig,
		Registers: *registersConfig,
	}, nil
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
