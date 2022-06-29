package util

import (
	"fmt"
	"golang.org/x/exp/constraints"
)

func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	} else {
		return b
	}
}

func NumericToGeneric[R float64 | uint16](v interface{}) R {
	switch v.(type) {
	case float64:
		return R(v.(float64))
	case int:
		return R(v.(int))
	case int64:
		return R(v.(int64))
	}
	panic(fmt.Sprintf("value %v is not numeric", v))
}
