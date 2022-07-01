package config

import (
	"fmt"
	"github.com/antonmedv/expr/vm"
	"gopkg.in/yaml.v3"
	"sungrow-prometheus-exporter/src/util"
)

type compilerFunc func(input string) (*vm.Program, error)

func expectOneElementMap[R any](m map[string]string, consumer func(key, value string) (R, error)) (R, error) {
	if len(m) != 1 {
		var n R
		return n, typeError("expecting map to contain exactly one element")
	}
	return consumer(util.GetOnlyMapElement(m))
}

func convertOneElementMapToFunction[X any](
	m map[string]string,
	compiler compilerFunc,
	envEntries ...*util.EnvEntry,
) (func(X) float64, error) {
	return expectOneElementMap(m, func(varName, expression string) (func(X) float64, error) {
		program, err := compiler(expression)
		if err != nil {
			return nil, err
		}
		return func(value X) float64 {
			result, err := vm.Run(program, util.BuildEnv(util.Env(varName, value).And(envEntries)...))
			util.PanicOnError(err)
			return util.NumericToFloat64(result)
		}, nil
	})
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
