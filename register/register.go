package register

import (
	"fmt"
	"reflect"
	"sungrow-prometheus-exporter/config"
)

type Reader func(address, quantity uint16) ([]uint16, error)

type Register interface {
	ReadWith(reader Reader) (Value, error)
}

type Value interface {
	AsString() string
	AsFloat64() float64
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
	quantity := uint16(reflect.TypeOf(T(0)).Size() / reflect.TypeOf(uint16(0)).Size())
	return &integerRegister{
		register{address: registerConfig.Address},
		mappers{
			mapToInt64: func(data []uint16) int64 {
				result := T(0)
				for i := uint16(0); i < quantity; i++ {
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
		quantity,
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
	quantity uint16
}

type integerValue struct {
	mappers
	data []uint16
}

func (value integerValue) AsString() string {
	return value.mapToString(value.asInt64())
}

func (value integerValue) AsFloat64() float64 {
	return value.mapToFloat64(value.asInt64())
}

func (value integerValue) asInt64() int64 {
	return value.mapToInt64(value.data)
}

func (register integerRegister) ReadWith(reader Reader) (Value, error) {
	data, err := reader(register.address, register.quantity)
	if err != nil {
		return nil, err
	}
	return &integerValue{register.mappers, data}, nil
}
