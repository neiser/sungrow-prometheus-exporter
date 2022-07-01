package config

import (
	"gopkg.in/yaml.v3"
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

func (m Metric) GetKey() string {
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
			r = append(r, expressionValue.registerNames...)
		}
	}
	return r
}
