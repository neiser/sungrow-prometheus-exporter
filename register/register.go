package register

import (
	"fmt"
	"reflect"
	"sungrow-prometheus-exporter/config"
)

type Reader interface {
	Read(address, quantity uint16) ([]uint16, error)
}

type Register interface {
	ReadWith(reader Reader, index uint16) (Value, error)
}

type Value interface {
	fmt.Stringer
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
	case config.StringRegisterType:
		return newStringRegister(registerConfig)
	}
	panic(fmt.Sprintf("unknown register type '%s'", registerConfig.Type))
}

func newStringRegister(registerConfig *config.Register) *stringRegister {
	return &stringRegister{register{
		registerConfig.Address,
		registerConfig.Length,
	}}
}

type stringRegister struct {
	register
}

type stringValue struct {
	data []uint16
}

func (s stringValue) String() string {
	var result []byte
	for i := 0; i < len(s.data); i++ {
		if b := byte(s.data[i] >> 8); b != 0 {
			result = append(result, b)
		}
		if b := byte(s.data[i] & 0xFF); b != 0 {
			result = append(result, b)
		}
	}
	return string(result)
}

func (s stringValue) AsFloat64() float64 {
	panic("string value does not have float64 representation")
}

func (r stringRegister) ReadWith(reader Reader, index uint16) (Value, error) {
	data, err := reader.Read(r.address, r.width)
	if err != nil {
		return nil, err
	}
	return &stringValue{data: data}, nil
}

func newIntegerRegister[T uint16 | uint32 | int16 | int32](registerConfig *config.Register) *integerRegister {
	width := uint16(reflect.TypeOf(T(0)).Size() / reflect.TypeOf(uint16(0)).Size())
	return &integerRegister{
		register{
			registerConfig.Address,
			width,
		},
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
	}

}

type register struct {
	address uint16
	width   uint16
}

type mappers struct {
	mapToInt64   func(data []uint16) int64
	mapToFloat64 func(value int64) float64
	mapToString  func(value int64) string
}

type integerRegister struct {
	register
	mappers
}

type integerValue struct {
	mappers
	data []uint16
}

func (v integerValue) String() string {
	return v.mapToString(v.mapToInt64(v.data))
}

func (v integerValue) AsFloat64() float64 {
	return v.mapToFloat64(v.mapToInt64(v.data))
}

func (r integerRegister) ReadWith(reader Reader, index uint16) (Value, error) {
	data, err := reader.Read(r.address+index*r.width, r.width)
	if err != nil {
		return nil, err
	}
	return &integerValue{r.mappers, data}, nil
}
