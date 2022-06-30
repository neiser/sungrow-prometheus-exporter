package config

import (
	"fmt"
	"github.com/antonmedv/expr/vm"
	"gopkg.in/yaml.v3"
	"sungrow-prometheus-exporter/src/util"
)

func convertOneElementMapToFunction[X any, Y any](
	m map[string]string,
	compiler func(input string) (*vm.Program, error),
	converter func(interface{}) Y,
	envEntries ...*util.EnvEntry,
) (func(X) Y, error) {
	if len(m) != 1 {
		return nil, typeError("expecting mapValue to contain exactly one element")
	}
	varName, expression := util.GetOnlyMapElement(m)
	program, err := compiler(expression)
	if err != nil {
		return nil, err
	}
	return func(value X) Y {
		result, err := vm.Run(program, util.BuildEnv(util.Env(varName, value).And(envEntries)...))
		if err != nil {
			panic(err.Error())
		}
		return converter(result)
	}, nil
}

func unmarshalNamedSequenceToMap[K util.HasKey](node *yaml.Node, result *map[string]*K) error {
	var s []K
	err := node.Decode(&s)
	if err != nil {
		return err
	}
	*result = util.MapFromNamedSlice(func(item K) *K {
		return &item
	}, s...)
	return nil
}

func typeError(msg string, a ...any) error {
	return &yaml.TypeError{Errors: []string{fmt.Sprintf(msg, a)}}
}
