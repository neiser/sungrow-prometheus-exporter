package util

import (
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInvertAndCompile(t *testing.T) {
	expressions := []string{
		"x+5",
		"x-5",
		"3/x",
		"x/0.1",
		"3.1*x",
		"x*3.2",
		"1/(x+5)",
		"1/(x-5)",
		"3 - (7 - 1/(5*x)) + 8",
	}
	for _, expression := range expressions {
		t.Run(expression, func(t *testing.T) {
			f, err := expr.Compile(expression)
			if err != nil {
				t.Fatal(err)
			}
			fInverse, err := InvertAndCompile(expression)
			if err != nil {
				t.Fatal(err)
			}
			for _, x := range []float64{1.0, 2.0, 5.0, -5.0} {
				y, err := vm.Run(f, map[string]float64{"x": x})
				if err != nil {
					t.Fatal(err)
				}
				xInverse, err := vm.Run(fInverse, map[string]float64{"x": NumericToFloat64(y)})
				if err != nil {
					t.Fatal(err)
				}
				assert.InDelta(t, x, xInverse, 0.00001)
			}
		})
	}
}
