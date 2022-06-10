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
	ReadFloat64(reader Reader, index uint16) (float64, error)
	ReadString(reader Reader) (string, error)
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

type register struct {
	address uint16
	width   uint16
}

type stringRegister struct {
	register
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

func newStringRegister(registerConfig *config.Register) *stringRegister {
	return &stringRegister{register{
		registerConfig.Address,
		registerConfig.Length,
	}}
}

func (r *stringRegister) ReadFloat64(reader Reader, index uint16) (float64, error) {
	panic("string register does not have float64 representation")
}

func (r *stringRegister) ReadString(reader Reader) (string, error) {
	data, err := reader.Read(r.address, r.width)
	if err != nil {
		return "", err
	}
	return convertDataToString(data), nil
}

func convertDataToString(data []uint16) string {
	var result []byte
	for i := 0; i < len(data); i++ {
		if b := byte(data[i] >> 8); b != 0 {
			result = append(result, b)
		}
		if b := byte(data[i] & 0xFF); b != 0 {
			result = append(result, b)
		}
	}
	return string(result)
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

func (r *integerRegister) ReadString(reader Reader) (string, error) {
	data, err := reader.Read(r.address, r.width)
	if err != nil {
		return "", err
	}
	return r.mapToString(r.mapToInt64(data)), nil
}

func (r *integerRegister) ReadFloat64(reader Reader, index uint16) (float64, error) {
	data, err := reader.Read(r.address+index*r.width, r.width)
	if err != nil {
		return 0, err
	}
	return r.mapToFloat64(r.mapToInt64(data)), nil
}
