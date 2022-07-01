package config

import (
	"gopkg.in/yaml.v3"
	"sungrow-prometheus-exporter/src/util"
	"time"
)

type ExpressionValue struct {
	registerFunc
}

func (v *ExpressionValue) UnmarshalYAML(node *yaml.Node) error {
	s := ""
	err := node.Decode(&s)
	if err != nil {
		return err
	}
	regFunc, err := newRegisterFunc(s, util.Env(
		"timeDate", func(args ...interface{}) (interface{}, error) {
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
		}))
	if err != nil {
		return err
	}
	*v = ExpressionValue{*regFunc}
	return nil
}

func (v *ExpressionValue) Evaluate(registerValue RegisterValueProvider) (interface{}, error) {
	return v.registerFunc.evaluate(registerValue)
}
