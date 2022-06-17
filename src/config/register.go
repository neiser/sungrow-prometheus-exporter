package config

import (
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"sungrow-prometheus-exporter/src/util"
)

type Registers map[string]*Register

func (registers *Registers) UnmarshalYAML(node *yaml.Node) error {
	return unmarshalSequenceToMap[Register](node, (*map[string]*Register)(registers))
}

type Register struct {
	Name     string       `yaml:"name"`
	Type     RegisterType `yaml:"type"`
	Address  uint16       `yaml:"address"`
	Writable bool         `yaml:"writable"`
	Length   uint16       `yaml:"length"`
	Unit     string       `yaml:"unit"`
	MapValue MapValue     `yaml:"mapValue"`
}

func (m Register) getName() string {
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

type MapValue struct {
	ByFunction func(value int64) float64
	ByEnumMap  map[int64]string
}

func (mapValue *MapValue) UnmarshalYAML(node *yaml.Node) error {
	m := map[string]string{}
	err := node.Decode(m)
	if err != nil {
		return err
	}
	switch len(m) {
	case 0:
		return &yaml.TypeError{Errors: []string{"mapValue should not be empty"}}
	case 1:
		{
			for x, y := range m {
				if len(x) == 1 {
					program, err := expr.Compile(y)
					if err != nil {
						log.Warnf("Ignoring uncompilable expression '%s', will assume one-element enum map, caused by: %s", y, err.Error())
						continue
					}
					mapValue.ByFunction = func(value int64) float64 {
						result, err := vm.Run(program, map[string]interface{}{x: value})
						if err != nil {
							panic(err.Error())
						}
						return util.NumericToFloat64(result)
					}
					return nil
				}
			}
		}
		fallthrough
	default:
		return node.Decode(&mapValue.ByEnumMap)
	}
}
