package config

import (
	"fmt"
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"gopkg.in/yaml.v3"
	"sungrow-prometheus-exporter/src/util"
)

func convertOneElementMapToFunction(m map[string]string) (func(int64) float64, error) {
	if len(m) != 1 {
		return nil, typeError("expecting mapValue to contain exactly one element")
	}
	varName, expression := util.GetOnlyMapElement(m)
	program, err := expr.Compile(expression)
	if err != nil {
		return nil, err
	}
	return func(value int64) float64 {
		result, err := vm.Run(program, map[string]interface{}{varName: value})
		if err != nil {
			panic(err.Error())
		}
		return util.NumericToFloat64(result)
	}, nil
}

type named interface {
	getName() string
}

func unmarshalNamedSequenceToMap[T named](node *yaml.Node, result *map[string]*T) error {
	var s []*T
	err := node.Decode(&s)
	if err != nil {
		return err
	}
	*result = make(map[string]*T)
	for _, metric := range s {
		(*result)[(*metric).getName()] = metric
	}
	return nil
}

func typeError(msg string, a ...any) error {
	return &yaml.TypeError{Errors: []string{fmt.Sprintf(msg, a)}}
}
