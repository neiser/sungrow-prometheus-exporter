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
	ByFunction func(value string) float64
}

func (mapValue *ActuatorMapValue) UnmarshalYAML(node *yaml.Node) error {
	m := map[string]string{}
	err := node.Decode(m)
	if err != nil {
		return err
	}
	if len(m) == 1 {
		function, err := convertOneElementMapToFunction[string](m, util.Compile, util.NumericToFloat64,
			util.Env("timeParse", func(value, layout, timezone string) time.Time {
				parse, err := time.Parse(layout, value)
				util.PanicOnError(err)
				location, err := time.LoadLocation(timezone)
				util.PanicOnError(err)
				return parse.In(location)
			}),
		)
		if err != nil {
			return err
		}
		mapValue.ByFunction = function

	}
	return nil
}
