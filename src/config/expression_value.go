package config

import (
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"sungrow-prometheus-exporter/src/util"
	"time"
)

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
	env := util.BuildEnv(
		util.Env(
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
			}),
		util.Env("register", func(args ...interface{}) (interface{}, error) {
			registerName := args[0].(string)
			registerValue, found := registers[registerName]
			if !found {
				registers[registerName] = 0
			}
			return registerValue, nil
		}))
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
