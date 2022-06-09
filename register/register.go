package register

import (
	"fmt"
	configPkg "sungrow-prometheus-exporter/config"
)

type Reader func(address, quantity uint16) ([]byte, error)

type Register interface {
	ReadWith(reader Reader) (Value, error)
}

type Value interface {
	AsString() string
	AsFloat64() float64
}

func New(config *configPkg.RegisterValue) Register {
	switch config.Type {
	case configPkg.U16RegisterType:
		return newIntegerRegister(config, func(get getByte) int64 {
			return int64(twoBytesAsInt[uint16](0, get))
		})
	case configPkg.U32RegisterType:
		return newIntegerRegister(config, func(get getByte) int64 {
			return int64(twoBytesAsInt[uint32](0, get) + twoBytesAsInt[uint32](2, get)<<16)
		})
	}
	panic("unknown register type")
}

type getByte func(i int) byte

func twoBytesAsInt[T uint16 | uint32](offset int, get getByte) T {
	return T(get(offset+1)) + T(get(offset+0))<<8
}

func newIntegerRegister(config *configPkg.RegisterValue, mapToInt64 func(bytes getByte) int64) *integerRegister {
	maxIndex := 0
	mapToInt64(func(i int) byte {
		if i > maxIndex {
			maxIndex = i
		}
		return 0
	})
	quantity := uint16(maxIndex+1) / 2
	return &integerRegister{
		register{address: config.Address},
		mapper{
			mapToInt64: func(bytes []byte) int64 {
				return mapToInt64(func(i int) byte {
					return bytes[i]
				})
			},
			mapToFloat64: func(value int64) float64 {
				if functionMapper, ok := config.MapValue().(*configPkg.FunctionMapValue); ok {
					return functionMapper.Map(value)
				}
				return float64(value)
			},
			mapToString: func(value int64) string {
				if enumMapper, ok := config.MapValue().(*configPkg.EnumMapValue); ok {
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
	mapToInt64   func(bytes []byte) int64
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
	bytes []byte
}

func (value integerValue) AsString() string {
	return value.mapToString(value.asInt64())
}

func (value integerValue) AsFloat64() float64 {
	return value.mapToFloat64(value.asInt64())
}

func (value integerValue) asInt64() int64 {
	return value.mapToInt64(value.bytes)
}

func (register integerRegister) ReadWith(reader Reader) (Value, error) {
	bytes, err := reader(register.address, register.quantity)
	if err != nil {
		return nil, err
	}
	return &integerValue{register.mapper, bytes}, nil
}
