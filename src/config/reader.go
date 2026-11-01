package config

import (
	"io/fs"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Metrics   Metrics
	Registers Registers
	Actuators Actuators
}

func Read(files fs.FS) (*Config, error) {
	metrics, err := unmarshalFromFile[Metrics](files, "metrics.yaml")
	if err != nil {
		return nil, err
	}
	registers, err := unmarshalFromFile[Registers](files, "registers.yaml")
	if err != nil {
		return nil, err
	}
	actuators, err := unmarshalFromFile[Actuators](files, "actuators.yaml")
	if err != nil {
		return nil, err
	}
	return &Config{*metrics, *registers, *actuators}, nil
}

func unmarshalFromFile[T any](files fs.FS, filename string) (*T, error) {
	file, err := fs.ReadFile(files, filename)
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
