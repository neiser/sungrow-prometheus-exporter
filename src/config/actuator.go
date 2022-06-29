package config

import (
	"gopkg.in/yaml.v3"
	"sungrow-prometheus-exporter/src/util"
	"time"
)

type Actuators map[string]*Actuator

func (actuators *Actuators) UnmarshalYAML(node *yaml.Node) error {
	return unmarshalNamedSequenceToMap[Actuator](node, (*map[string]*Actuator)(actuators))
}

type Actuator struct {
	Name      string `yaml:"name"`
	Registers map[string]ActuatorMapValue
}

func (a Actuator) GetKey() string {
	return a.Name
}

type ActuatorMapValue struct {
	ByFunction func(value string) uint16
}

func (mapValue *ActuatorMapValue) UnmarshalYAML(node *yaml.Node) error {
	m := map[string]string{}
	err := node.Decode(m)
	if err != nil {
		return err
	}
	if len(m) == 1 {
		function, err := convertOneElementMapToFunction[string, uint16](m,
			util.Env("timeParse", func(value, layout string) time.Time {
				parse, err := time.Parse(layout, value)
				if err != nil {
					panic(err.Error())
				}
				return parse
			}),
		)
		if err != nil {
			return err
		}
		mapValue.ByFunction = function

	}
	return nil
}
