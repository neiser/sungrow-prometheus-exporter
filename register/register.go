package register

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"reflect"
	"sungrow-prometheus-exporter/config"
)

type Reader func(address, quantity uint16) ([]uint16, error)

type Register interface {
	ReadWith(reader Reader) (Value, error)
}

type Value interface {
	AsString() string
	AsFloat64s() []float64
}

func NewFromConfig(registerConfig *config.Register) Register {
	switch registerConfig.Type {
	case config.U16RegisterType:
		return newIntegerRegister[uint16](registerConfig)
	case config.U32RegisterType:
		return newIntegerRegister[uint32](registerConfig)
	case config.S16RegisterType:
		return newIntegerRegister[int16](registerConfig)
	case config.S32RegisterType:
		return newIntegerRegister[int32](registerConfig)
	}
	panic("unknown register type")
}

func newIntegerRegister[T uint16 | uint32 | int16 | int32](registerConfig *config.Register) *integerRegister {
	width := uint16(reflect.TypeOf(T(0)).Size() / reflect.TypeOf(uint16(0)).Size())
	length := 1
	if registerConfig.Length > 1 {
		length = registerConfig.Length
	}
	return &integerRegister{
		register{address: registerConfig.Address},
		mappers{
			mapToInt64: func(data []uint16) int64 {
				result := T(0)
				for i := uint16(0); i < width; i++ {
					result += T(data[i]) << (16 * i)
				}
				return int64(result)
			},
			mapToFloat64: func(value int64) float64 {
				if mapper := registerConfig.MapValue.ByFunction; mapper != nil {
					return mapper(value)
				}
				return float64(value)
			},
			mapToString: func(value int64) string {
				if mapper := registerConfig.MapValue.ByEnumMap; mapper != nil {
					if mappedValue, ok := mapper[value]; ok {
						return mappedValue
					}
				}
				return fmt.Sprintf("%d", value)
			},
		},
		width,
		uint16(length),
	}

}

type register struct {
	address uint16
}

type mappers struct {
	mapToInt64   func(data []uint16) int64
	mapToFloat64 func(value int64) float64
	mapToString  func(value int64) string
}

type integerRegister struct {
	register
	mappers
	width  uint16
	length uint16
}

type integerValue struct {
	mappers
	slicedData [][]uint16
}

func (value integerValue) AsString() string {
	if len(value.slicedData) != 1 {
		panic("cannot handle sliced data as string")
	}
	return value.mapToString(value.mapToInt64(value.slicedData[0]))
}

func (value integerValue) AsFloat64s() []float64 {
	result := make([]float64, len(value.slicedData))
	for i := 0; i < len(result); i++ {
		result[i] = value.mapToFloat64(value.mapToInt64(value.slicedData[i]))
	}
	return result
}

func (register integerRegister) ReadWith(reader Reader) (Value, error) {
	quantity := register.width * register.length
	log.Infof("Reading %d from address %d", quantity, register.address)
	data, err := reader(register.address, quantity)
	if err != nil {
		return nil, err
	}
	slicedData := make([][]uint16, register.length)
	for i := uint16(0); i < register.length; i++ {
		slicedData[i] = data[register.width*i : register.width*(i+1)]
	}
	return &integerValue{register.mappers, slicedData}, nil
}
