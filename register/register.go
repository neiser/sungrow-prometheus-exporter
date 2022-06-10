package register

import (
	"fmt"
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
		return newIntegerRegister(registerConfig, func(get getData) int64 {
			return int64(get(0))
		})
	case config.U32RegisterType:
		return newIntegerRegister(registerConfig, func(get getData) int64 {
			return int64(uint32(get(0)) + uint32(get(1))<<16)
		})
	}
	panic("unknown register type")
}

type getData func(i int) uint16

func newIntegerRegister(registerConfig *config.Register, mapToInt64 func(data getData) int64) *integerRegister {
	maxIndex := 0
	mapToInt64(func(i int) uint16 {
		if i > maxIndex {
			maxIndex = i
		}
		return 0
	})
	quantity := uint16(maxIndex + 1)
	return &integerRegister{
		register{address: registerConfig.Address},
		mapper{
			mapToInt64: func(data []uint16) int64 {
				return mapToInt64(func(i int) uint16 {
					return data[i]
				})
			},
			mapToFloat64: func(value int64) float64 {
				if functionMapper := registerConfig.MapValue.FunctionMapValue; functionMapper != nil {
					return functionMapper.Map(value)
				}
				return float64(value)
			},
			mapToString: func(value int64) string {
				if enumMapper := registerConfig.MapValue.EnumMapValue; enumMapper != nil {
					return enumMapper.Map(value)
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

type mapper struct {
	mapToInt64   func(data []uint16) int64
	mapToFloat64 func(value int64) float64
	mapToString  func(value int64) string
}

type integerRegister struct {
	register
	mapper
	quantity uint16
}

type integerValue struct {
	mapper
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
	return &integerValue{register.mapper, data}, nil
}
