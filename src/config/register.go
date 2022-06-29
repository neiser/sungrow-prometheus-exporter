package config

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Registers map[string]*Register

func (registers *Registers) UnmarshalYAML(node *yaml.Node) error {
	return unmarshalNamedSequenceToMap[Register](node, (*map[string]*Register)(registers))
}

type RegisterValidation func(float64) bool

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

type RegisterMapValue struct {
	ByFunction func(value int64) float64
	ByEnumMap  map[int64]string
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
			function, err := convertOneElementMapToFunction[int64, float64](m)
			if err == nil {
				mapValue.ByFunction = function
				return nil
			}
			log.Warnf("Assuming one-element enum map, caused by: %s", err.Error())
		}
		fallthrough
	default:
		return node.Decode(&mapValue.ByEnumMap)
	}
}
