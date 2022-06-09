package config

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/Knetic/govaluate.v3"
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

func (getter *ValueGetter) UnmarshalYAML(node *yaml.Node) error {
	var m map[ValueType]interface{}
	err := node.Decode(&m)
	if err != nil {
		return err
	}
	if len(m) != 1 {
		return &yaml.TypeError{Errors: []string{"expected exactly one key"}}
	}
	if value, ok := m[ExpressionValueType]; ok {
		expression := fmt.Sprintf("%v", value)
		evaluableExpression, err := govaluate.NewEvaluableExpression(expression)
		if err != nil {
			return errors.Wrapf(err, "cannot parse '%s'", expression)
		}
		*getter = func() Value {
			return &ExpressionValue{Expression: evaluableExpression}
		}
		return nil
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
	return &yaml.TypeError{Errors: []string{"unknown value type"}}
}

type ExpressionValue struct {
	Expression *govaluate.EvaluableExpression
}

type RegisterValue struct {
	Type     RegisterType   `yaml:"type"`
	Address  uint16         `yaml:"address"`
	Length   int            `yaml:"length"`
	Interval string         `yaml:"interval"`
	MapValue MapValueGetter `yaml:"mapValue"`
}

type MapValue interface {
}

type MapValueGetter func() MapValue

type FunctionMapValue struct {
	Map func(value int64) float64
}

type EnumMapValue struct {
	Map func(value int64) string
}

func (getter *MapValueGetter) UnmarshalYAML(node *yaml.Node) error {
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
					expression, err := govaluate.NewEvaluableExpression(y)
					if err != nil {
						continue
					}
					*getter = func() MapValue {
						return &FunctionMapValue{Map: func(value int64) float64 {
							y, _ := expression.Evaluate(map[string]interface{}{x: value})
							return y.(float64)
						}}
					}
					return nil
				}
			}
		}
		fallthrough
	default:
		{
			enumMap := make(map[int64]string)
			for x, y := range m {
				intExpr, err := govaluate.NewEvaluableExpression(x)
				if err != nil {
					return errors.Wrapf(err, "cannot parse '%s' as integer expression", x)
				}
				result, err := intExpr.Evaluate(map[string]interface{}{})
				if err != nil {
					return errors.Wrapf(err, "cannot eval '%s' as integer expression", x)
				}
				if floatValue, ok := result.(float64); ok {
					intValue := int64(floatValue)
					if _, contains := enumMap[intValue]; contains {
						return &yaml.TypeError{Errors: []string{"duplicate key for enum map"}}
					}
					enumMap[intValue] = y
				} else {
					return &yaml.TypeError{Errors: []string{"expression did not eval to int value"}}
				}
			}
			*getter = func() MapValue {
				return &EnumMapValue{Map: func(value int64) string {
					return enumMap[value]
				}}
			}
			return nil
		}
	}
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
