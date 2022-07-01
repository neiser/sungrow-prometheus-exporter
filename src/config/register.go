package config

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"sungrow-prometheus-exporter/src/util"
)

type Registers map[string]*Register

func (registers *Registers) UnmarshalYAML(node *yaml.Node) error {
	return unmarshalNamedSequenceToMap[Register](node, (*map[string]*Register)(registers))
}

type Register struct {
	Name       string             `yaml:"name"`
	Type       RegisterType       `yaml:"type"`
	Address    uint16             `yaml:"address"`
	Writable   bool               `yaml:"writable"`
	Validation RegisterValidation `yaml:"validation"`
	Length     uint16             `yaml:"length"`
	Unit       string             `yaml:"unit"`
	MapValue   RegisterMapValue   `yaml:"mapValue"`
}

func (m Register) GetKey() string {
	return m.Name
}

type RegisterType string

const (
	U16RegisterType    RegisterType = "u16"
	U32RegisterType    RegisterType = "u32"
	S16RegisterType    RegisterType = "s16"
	S32RegisterType    RegisterType = "s32"
	StringRegisterType RegisterType = "string"
)

type RegisterValidation func(value float64, provider RegisterValueProvider) error

func (validation *RegisterValidation) UnmarshalYAML(node *yaml.Node) error {
	m := map[string]string{}
	err := node.Decode(m)
	if err != nil {
		return err
	}
	*validation, err = expectOneElementMap(m, func(varName, expression string) (RegisterValidation, error) {
		return func(value float64, provider RegisterValueProvider) error {
			regFunc, err := newRegisterFunc(expression, util.Env(varName, value))
			if err != nil {
				return err
			}
			valid, err := regFunc.evaluate(provider)
			if err != nil {
				return err
			}
			if !util.CastToBool(valid) {
				return fmt.Errorf("invalid value '%f'", value)
			}
			return nil
		}, nil
	})
	return err
}

type RegisterMapValue struct {
	ByFunction         func(value int64) float64
	GetInverseFunction func() (func(float64) float64, error)
	ByEnumMap          map[int64]string
}

func (mapValue *RegisterMapValue) UnmarshalYAML(node *yaml.Node) error {
	m := map[string]string{}
	err := node.Decode(m)
	if err != nil {
		return err
	}
	switch len(m) {
	case 0:
		return typeError("mapValue should not be empty")
	case 1:
		{
			function, err := convertOneElementMapToFunction[int64](m, util.Compile)
			if err == nil {
				mapValue.ByFunction = function
				mapValue.GetInverseFunction = func() (func(float64) float64, error) {
					return convertOneElementMapToFunction[float64](m, util.InvertAndCompile)
				}
				return nil
			}
			log.Warnf("Assuming one-element enum map instead of function, caused by: %s", err.Error())
		}
		fallthrough
	default:
		return node.Decode(&mapValue.ByEnumMap)
	}
}
