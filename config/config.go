package config

import (
	"github.com/pkg/errors"
	"gopkg.in/Knetic/govaluate.v3"
	"gopkg.in/yaml.v3"
	"os"
	"strconv"
	"strings"
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
	Name   string     `yaml:"name"`
	Help   string     `yaml:"help"`
	Type   MetricType `yaml:"type"`
	Value  Value      `yaml:"value"`
	Labels []*Label   `yaml:"labels"`
}

type Label struct {
	Name  string `yaml:"name"`
	Value Value  `yaml:"value"`
}

type Value struct {
	FromExpression *ExpressionValue `yaml:"fromExpression"`
	FromRegister   *Register        `yaml:"fromRegister"`
}

func (expressionValue *ExpressionValue) UnmarshalYAML(node *yaml.Node) error {
	s := ""
	err := node.Decode(&s)
	if err != nil {
		return err
	}
	expression, err := govaluate.NewEvaluableExpression(s)
	if err != nil {
		return errors.Wrapf(err, "cannot parse '%s' as expression", s)
	}
	*expressionValue = ExpressionValue{expression}
	return nil
}

type ExpressionValue struct {
	*govaluate.EvaluableExpression
}

type Register struct {
	Type     RegisterType `yaml:"type"`
	Address  uint16       `yaml:"address"`
	Length   int          `yaml:"length"`
	Interval string       `yaml:"interval"`
	MapValue MapValue     `yaml:"mapValue"`
}

type MapValue struct {
	FunctionMapValue *FunctionMapValue
	EnumMapValue     *EnumMapValue
}

type FunctionMapValue struct {
	Map func(value int64) float64
}

type EnumMapValue struct {
	Map func(value int64) string
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
					expression, err := govaluate.NewEvaluableExpression(y)
					if err != nil {
						continue
					}
					mapValue.FunctionMapValue = &FunctionMapValue{Map: func(value int64) float64 {
						y, _ := expression.Evaluate(map[string]interface{}{x: value})
						return y.(float64)
					}}
					return nil
				}
			}
		}
		fallthrough
	default:
		{
			enumMap := make(map[int64]string)
			for x, y := range m {
				intValue, err := parseInt(x)
				if err != nil {
					return errors.Wrapf(err, "cannot parse '%s' as integer", x)
				}
				enumMap[intValue] = y
			}
			mapValue.EnumMapValue = &EnumMapValue{Map: func(value int64) string {
				return enumMap[value]
			}}
			return nil
		}
	}
}

func parseInt(s string) (int64, error) {
	base := 10
	if strings.HasPrefix(s, "0x") {
		s = s[2:]
		base = 16

	}
	result, err := strconv.ParseInt(s, base, 64)
	if err != nil {
		return 0, err
	}
	return result, nil
}

type MetricType string
type RegisterType string

const (
	Gauge   MetricType = "gauge"
	Counter MetricType = "counter"

	U16RegisterType    RegisterType = "u16"
	U32RegisterType    RegisterType = "u32"
	S16RegisterType    RegisterType = "s16"
	S32RegisterType    RegisterType = "s32"
	StringRegisterType RegisterType = "string"
)
