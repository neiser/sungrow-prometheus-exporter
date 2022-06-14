package register

import (
	"fmt"
	"reflect"
	"strconv"
	"sungrow-prometheus-exporter/src/config"
	"sungrow-prometheus-exporter/src/util"
)

type Reader func(address, quantity uint16, writable bool) ([]uint16, error)

type Register interface {
	ReadFloat64(reader Reader, index uint16) (float64, error)
	ReadString(reader Reader) (string, error)
	getAddressInterval() *util.Interval[uint16]
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

func FindAddressIntervals(registerNames []string, registerConfigs config.Registers) (readAddressIntervals util.Intervals[uint16], writeAddressIntervals util.Intervals[uint16]) {
	for _, registerName := range registerNames {
		registerConfig := registerConfigs[registerName]
		if registerConfig.Writable {
			writeAddressIntervals = append(writeAddressIntervals, NewFromConfig(registerConfig).getAddressInterval())
		} else {
			readAddressIntervals = append(readAddressIntervals, NewFromConfig(registerConfig).getAddressInterval())
		}
	}
	return
}

type register struct {
	baseAddress uint16
	width       uint16
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
	length   uint16
	writable bool
}

func newStringRegister(registerConfig *config.Register) *stringRegister {
	return &stringRegister{register{
		registerConfig.Address,
		registerConfig.Length,
	}}
}

func (r *stringRegister) getAddressInterval() *util.Interval[uint16] {
	panic("not implemented")
}

func (r *stringRegister) ReadFloat64(Reader, uint16) (float64, error) {
	panic("string register does not have float64 representation")
}

func (r *stringRegister) ReadString(reader Reader) (string, error) {
	data, err := reader(r.baseAddress, r.width, false)
	if err != nil {
		return "", err
	}
	return mapToString(data), nil
}

func mapToString(data []uint16) string {
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
	quantity := uint16(reflect.TypeOf(T(0)).Size() / reflect.TypeOf(uint16(0)).Size())
	length := uint16(1)
	if registerConfig.Length > 1 {
		length = registerConfig.Length
	}
	return &integerRegister{
		register{
			registerConfig.Address,
			quantity,
		},
		mappers{
			mapToInt64: func(data []uint16) int64 {
				result := T(0)
				for i := uint16(0); i < quantity; i++ {
					result += T(data[i]) << (16 * i)
				}
				return int64(result)
			},
			mapToFloat64: func(value int64) float64 {
				if mapper := registerConfig.MapValue.ByEnumMap; mapper != nil {
					if mappedValue, ok := mapper[value]; ok {
						convertedValue, err := strconv.ParseFloat(mappedValue, 64)
						if err == nil {
							return convertedValue
						}
					}
				}
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
				if mapper := registerConfig.MapValue.ByFunction; mapper != nil {
					return fmt.Sprintf("%v", mapper(value))
				}
				return fmt.Sprintf("%v", value)
			},
		},
		length,
		registerConfig.Writable,
	}
}

func (r *integerRegister) getAddressInterval() *util.Interval[uint16] {
	return &util.Interval[uint16]{r.baseAddress, r.baseAddress + (r.length-1)*r.width + (r.width - 1)}
}

func (r *integerRegister) ReadString(reader Reader) (string, error) {
	data, err := reader(r.baseAddress, r.width, r.writable)
	if err != nil {
		return "", err
	}
	return r.mapToString(r.mapToInt64(data)), nil
}

func (r *integerRegister) ReadFloat64(reader Reader, index uint16) (float64, error) {
	if index >= r.length {
		panic("register index out of range")
	}
	data, err := reader(r.baseAddress+index*r.width, r.width, r.writable)
	if err != nil {
		return 0, err
	}
	return r.mapToFloat64(r.mapToInt64(data)), nil
}
