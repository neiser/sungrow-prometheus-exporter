package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

func ReadFromFile(filename string) (*Config, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	config := &Config{}
	err = yaml.Unmarshal(file, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

type Config struct {
	Inverter *Inverter
	Metrics  []*Metric
}

type Inverter struct {
	Address string `yaml:"address"`
}

type Metric struct {
	Name   string      `yaml:"name"`
	Help   string      `yaml:"help"`
	Type   MetricType  `yaml:"type"`
	Value  ValueGetter `yaml:"value"`
	Labels []*Label
}

type Label struct {
	Name  string      `yaml:"name"`
	Value ValueGetter `yaml:"value"`
}

type Value interface {
}

type ValueGetter func() Value

type ExpressionValue struct {
	Expression string
}

func (getter *ValueGetter) UnmarshalYAML(node *yaml.Node) error {
	var m map[ValueType]interface{}
	err := node.Decode(&m)
	if err != nil {
		return err
	}
	if len(m) != 1 {
		return &yaml.TypeError{Errors: []string{"expected exactly one key"}}
	}
	if value, ok := m[RegisterValueType]; ok {
		registerValueBytes, err := yaml.Marshal(value)
		if err != nil {
			return err
		}
		registerValue := &RegisterValue{}
		err = yaml.Unmarshal(registerValueBytes, registerValue)
		if err != nil {
			return err
		}
		*getter = func() Value {
			return registerValue
		}
		return nil
	}
	if value, ok := m[ExpressionValueType]; ok {
		expression := fmt.Sprintf("%s", value)
		*getter = func() Value {
			return &ExpressionValue{Expression: expression}
		}
		return nil
	}
	return &yaml.TypeError{Errors: []string{"unknown value type"}}
}

type RegisterValue struct {
	Type     RegisterType `yaml:"type"`
	Address  uint16       `yaml:"address"`
	Length   int          `yaml:"length"`
	Interval string       `yaml:"interval"`
}

type MetricType string
type ValueType string
type RegisterType string

const (
	Gauge   MetricType = "gauge"
	Counter MetricType = "counter"

	ExpressionValueType ValueType = "fromExpression"
	RegisterValueType   ValueType = "fromRegister"

	U16RegisterType    RegisterType = "u16"
	U32RegisterType    RegisterType = "u32"
	S16RegisterType    RegisterType = "s16"
	S32RegisterType    RegisterType = "s32"
	StringRegisterType RegisterType = "string"
)
