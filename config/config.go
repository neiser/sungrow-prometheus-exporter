package config

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
	Name   string     `yaml:"name"`
	Help   string     `yaml:"help"`
	Type   MetricType `yaml:"type"`
	Value  *Value     `yaml:"value"`
	Labels []*Label   `yaml:"labels"`
}

type Label struct {
	Name  string `yaml:"name"`
	Value *Value `yaml:"value"`
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
	Length   uint16       `yaml:"length"`
	MapValue MapValue     `yaml:"mapValue"`
}

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
					expression, err := govaluate.NewEvaluableExpression(y)
					if err != nil {
						log.Warnf("Ignoring unparsable expression '%s' (caused by '%s'), will assume one-element enum map", y, err.Error())
						continue
					}
					mapValue.ByFunction = func(value int64) float64 {
						result, err := expression.Evaluate(map[string]interface{}{x: value})
						if err != nil {
							panic(err.Error())
						}
						return result.(float64)
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
