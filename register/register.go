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
	ReadWith(reader Reader) (Value, error)
}

type Value interface {
	fmt.Stringer
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
	case config.StringRegisterType:
		return newStringRegister(registerConfig)
	}
	panic(fmt.Sprintf("unknown register type '%s'", registerConfig.Type))
}

func newStringRegister(registerConfig *config.Register) *stringRegister {
	return &stringRegister{register{
		registerConfig.Address,
		1,
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

func (s stringValue) AsFloat64s() []float64 {
	panic("string value does not have float64 representation")
}

func (r stringRegister) ReadWith(reader Reader) (Value, error) {
	data, err := reader.Read(r.address, r.width*r.length)
	if err != nil {
		return nil, err
	}
	return &stringValue{data: data}, nil
}

func newIntegerRegister[T uint16 | uint32 | int16 | int32](registerConfig *config.Register) *integerRegister {
	width := uint16(reflect.TypeOf(T(0)).Size() / reflect.TypeOf(uint16(0)).Size())
	length := uint16(1)
	if registerConfig.Length > 1 {
		length = registerConfig.Length
	}
	return &integerRegister{
		register{
			registerConfig.Address,
			width,
			length,
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
	length  uint16
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
	slicedData [][]uint16
}

func (v integerValue) String() string {
	if len(v.slicedData) != 1 {
		panic("cannot handle sliced data as string")
	}
	return v.mapToString(v.mapToInt64(v.slicedData[0]))
}

func (v integerValue) AsFloat64s() []float64 {
	result := make([]float64, len(v.slicedData))
	for i := 0; i < len(result); i++ {
		result[i] = v.mapToFloat64(v.mapToInt64(v.slicedData[i]))
	}
	return result
}

func (r integerRegister) ReadWith(reader Reader) (Value, error) {
	data, err := reader.Read(r.address, r.width*r.length)
	if err != nil {
		return nil, err
	}
	slicedData := make([][]uint16, r.length)
	for i := uint16(0); i < r.length; i++ {
		slicedData[i] = data[r.width*i : r.width*(i+1)]
	}
	return &integerValue{r.mappers, slicedData}, nil
}
