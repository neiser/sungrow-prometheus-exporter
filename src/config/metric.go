package config

import (
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"time"
)

type Metrics map[string]*Metric

func (metrics *Metrics) UnmarshalYAML(node *yaml.Node) error {
	return unmarshalNamedSequenceToMap[Metric](node, (*map[string]*Metric)(metrics))
}

type Metric struct {
	Name   string     `yaml:"name"`
	Help   string     `yaml:"help"`
	Alias  string     `yaml:"alias"`
	Type   MetricType `yaml:"type"`
	Value  *Value     `yaml:"value"`
	Labels []*Label   `yaml:"labels"`
}

func (m Metric) getName() string {
	return m.Name
}

type MetricType string

const (
	Gauge   MetricType = "gauge"
	Counter MetricType = "counter"
)

type Label struct {
	Name  string `yaml:"name"`
	Value *Value `yaml:"value"`
}

type Value struct {
	FromExpression *ExpressionValue `yaml:"fromExpression"`
	FromRegister   *RegisterValue   `yaml:"fromRegister"`
}

type ExpressionValue struct {
	program   *vm.Program
	env       map[string]interface{}
	registers map[string]float64
}

func (v *ExpressionValue) UnmarshalYAML(node *yaml.Node) error {
	s := ""
	err := node.Decode(&s)
	if err != nil {
		return err
	}

	program, err := expr.Compile(s)
	if err != nil {
		return errors.Wrapf(err, "cannot compile '%s'", s)
	}
	registers := make(map[string]float64)
	env := map[string]interface{}{
		"timeDate": func(args ...interface{}) (interface{}, error) {
			location, err := time.LoadLocation(args[6].(string))
			if err != nil {
				return nil, err
			}
			return time.Date(
				int(args[0].(float64)),
				time.Month(args[1].(float64)),
				int(args[2].(float64)),
				int(args[3].(float64)),
				int(args[4].(float64)),
				int(args[5].(float64)),
				0,
				location,
			), nil
		},
		"register": func(args ...interface{}) (interface{}, error) {
			registerName := args[0].(string)
			registerValue, found := registers[registerName]
			if !found {
				registers[registerName] = 0
			}
			return registerValue, nil
		},
	}
	_, err = vm.Run(program, env)
	if err != nil {
		return errors.Wrapf(err, "cannot run '%s'", s)
	}
	*v = ExpressionValue{program, env, registers}
	return nil
}

func (v *ExpressionValue) Evaluate(registerValue func(registerName string) float64) (interface{}, error) {
	for registerName := range v.registers {
		v.registers[registerName] = registerValue(registerName)
	}
	result, err := vm.Run(v.program, v.env)
	if err != nil {
		return 0, err
	}
	return result, nil
}

type RegisterValue struct {
	Name string
}

func (v *RegisterValue) UnmarshalYAML(node *yaml.Node) error {
	s := ""
	err := node.Decode(&s)
	if err != nil {
		return err
	}
	*v = RegisterValue{s}
	return nil
}

func (metrics Metrics) FindRegisterNames() []string {
	var r []string
	for _, metric := range metrics {
		if registerValue := metric.Value.FromRegister; registerValue != nil {
			r = append(r, registerValue.Name)
		}
		if expressionValue := metric.Value.FromExpression; expressionValue != nil {
			for registerName := range expressionValue.registers {
				r = append(r, registerName)
			}
		}
	}
	return r
}
