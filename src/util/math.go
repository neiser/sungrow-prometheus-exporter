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

func NumericToFloat64(v interface{}) float64 {
	switch v.(type) {
	case float64:
		return v.(float64)
	case int:
		return float64(v.(int))
	case int64:
		return float64(v.(int64))
	}
	panic(fmt.Sprintf("value %v is not numeric", v))
}
