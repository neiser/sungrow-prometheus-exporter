package config

import (
	"gopkg.in/yaml.v3"
)

type Actuators map[string]*Actuator

func (actuators *Actuators) UnmarshalYAML(node *yaml.Node) error {
	return unmarshalNamedSequenceToMap[Actuator](node, (*map[string]*Actuator)(actuators))
}

type Actuator struct {
	Name      string `yaml:"name"`
	Registers map[string]ActuatorMapValue
}

func (a Actuator) getName() string {
	return a.Name
}

type ActuatorMapValue struct {
	ByFunction func(value int64) float64
}

func (mapValue *ActuatorMapValue) UnmarshalYAML(node *yaml.Node) error {
	m := map[string]string{}
	err := node.Decode(m)
	if err != nil {
		return err
	}
	if len(m) == 1 {
		function, err := convertOneElementMapToFunction(m)
		if err != nil {
			return err
		}
		mapValue.ByFunction = function

	}
	return nil
}
