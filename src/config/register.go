package config

import (
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

type RegisterValidation func(value float64) bool

func (validation *RegisterValidation) UnmarshalYAML(node *yaml.Node) error {
	m := map[string]string{}
	err := node.Decode(m)
	if err != nil {
		return err
	}
	if len(m) == 1 {
		function, err := convertOneElementMapToFunction[float64](m, util.Compile, util.CastToBool)
		if err != nil {
			return err
		}
		*validation = function
	}
	return nil
}

type RegisterMapValue struct {
	ByFunction         func(value int64) float64
	ByEnumMap          map[int64]string
	GetInverseFunction func() (func(float64) float64, error)
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
			function, err := convertOneElementMapToFunction[int64](m, util.Compile, util.NumericToFloat64)
			if err == nil {
				mapValue.ByFunction = function
				mapValue.GetInverseFunction = func() (func(float64) float64, error) {
					return convertOneElementMapToFunction[float64](m, util.InvertAndCompile, util.NumericToFloat64)
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
