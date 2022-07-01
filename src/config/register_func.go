package config

import (
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/pkg/errors"
	"sungrow-prometheus-exporter/src/util"
)

type RegisterValueProvider func(registerName string) float64

type registerFunc struct {
	registerNames []string
	evaluate      func(provider RegisterValueProvider) (interface{}, error)
}

func newRegisterFunc(input string, envs ...*util.EnvEntry) (*registerFunc, error) {
	program, err := expr.Compile(input)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot compile '%s'", input)
	}
	registerValues := make(map[string]float64)
	env := util.BuildEnv(
		util.Env("register", func(args ...interface{}) (interface{}, error) {
			registerName := args[0].(string)
			registerValue, found := registerValues[registerName]
			if !found {
				registerValues[registerName] = 0
			}
			return registerValue, nil
		}).And(envs)...)
	_, err = vm.Run(program, env)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot run '%s'", input)
	}
	return &registerFunc{util.GetKeys(registerValues), func(provider RegisterValueProvider) (interface{}, error) {
		for registerName := range registerValues {
			registerValues[registerName] = provider(registerName)
		}
		result, err := vm.Run(program, env)
		if err != nil {
			return 0, err
		}
		return result, nil
	}}, nil
}
