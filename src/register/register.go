package register

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
	"strconv"
	"strings"
	"sungrow-prometheus-exporter/src/config"
	"sungrow-prometheus-exporter/src/util"
)

type Reader interface {
	Read(address, quantity uint16, writable bool) ([]uint16, error)
}

type Writer interface {
	Write(address, quantity uint16, values []uint16) ([]uint16, error)
}

type ReadWriter interface {
	Reader
	Writer
}

type Register interface {
	ReadFloat64(reader Reader, index uint16) (float64, error)
	ReadString(reader Reader) (string, error)
	getAddressInterval() *util.Interval[uint16]
	getValueToWrite(valueProvider func() (string, *float64)) uint16
}

type Registers map[string]Register

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

func NewFromConfigs(registersConfig config.Registers, registerNames ...string) Registers {
	r := Registers{}
	for _, registerName := range registerNames {
		r[registerName] = NewFromConfig(registersConfig[registerName])
	}
	return r
}

func FindAddressIntervals(registerConfigs config.Registers, registerNames ...string) (readAddressIntervals util.Intervals[uint16], writeAddressIntervals util.Intervals[uint16]) {
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

type registerNameAndValue struct {
	registerName string
	value        uint16
}

func (r registerNameAndValue) getValue() uint16 {
	return r.value
}

func (r registerNameAndValue) getRegisterName() string {
	return r.registerName
}

type WrittenRegisterValues struct {
	registerSlices util.IntervalSlices[uint16, registerNameAndValue]
}

func (registers Registers) Write(writer Writer, valueProvider func(registerName string) (string, *float64)) (*WrittenRegisterValues, error) {

	registerSlices := util.IntervalSlices[uint16, registerNameAndValue]{}

	for registerName, reg := range registers {
		addressInterval := reg.getAddressInterval()
		if addressInterval.Length() != 1 {
			return nil, fmt.Errorf("cannot write into register %s with length != 1", registerName)
		}
		value := reg.getValueToWrite(func() (string, *float64) {
			return valueProvider(registerName)
		})

		registerSlices = append(registerSlices,
			util.NewIntervalSlice(addressInterval, registerNameAndValue{registerName, value}),
		)
	}

	registerSlices.SortAndMerge()

	for _, reg := range registerSlices {
		written, err := writer.Write(reg.Start, reg.Length(), util.MapSlice(reg.Slice, registerNameAndValue.getValue))
		if err != nil {
			return nil, err
		}
		for k, value := range written {
			reg.Slice[k].value = value
		}
	}
	return &WrittenRegisterValues{registerSlices}, nil
}

func (w WrittenRegisterValues) Read(startAddress, quantity uint16, writable bool) ([]uint16, error) {
	if !writable {
		panic("cannot read non-writable registers")
	}
	valuesByAddress := make(map[uint16]uint16)
	for _, reg := range w.registerSlices {
		for k, value := range util.MapSlice(reg.Slice, registerNameAndValue.getValue) {
			address := reg.Start + uint16(k)
			valuesByAddress[address] = value
		}
	}
	result := make([]uint16, quantity)
	for i := uint16(0); i < quantity; i++ {
		result[i] = valuesByAddress[startAddress+i]
	}
	return result, nil
}

func (w WrittenRegisterValues) String() string {
	var result []string
	for _, reg := range w.registerSlices {
		registerNames := util.MapSlice(reg.Slice, registerNameAndValue.getRegisterName)
		for k, value := range util.MapSlice(reg.Slice, registerNameAndValue.getValue) {
			result = append(result, fmt.Sprintf("%s=%d", registerNames[k], value))
		}
	}
	return strings.Join(result, ", ")
}

type register struct {
	baseAddress uint16
	width       uint16
}

type stringRegister struct {
	register
}

type mappers struct {
	mapToInt64     func(data []uint16) int64
	mapToFloat64   func(value int64) float64
	mapToString    func(value int64) string
	mapFromFloat64 func(value float64) uint16
	mapFromString  func(value string) uint16
}

type integerRegister struct {
	register
	mappers
	length   uint16
	writable bool
}

func (r *integerRegister) getValueToWrite(valueProvider func() (string, *float64)) uint16 {
	if !r.writable {
		panic("register is not writable")
	}
	if r.width != 1 {
		panic("writing register with width > 1 is not supported")
	}
	stringValue, floatValue := valueProvider()
	if floatValue != nil {
		return r.mapFromFloat64(*floatValue)
	}
	return r.mapFromString(stringValue)
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

func (r *stringRegister) getValueToWrite(func() (string, *float64)) uint16 {
	panic("cannot write string register")
}

func (r *stringRegister) ReadFloat64(Reader, uint16) (float64, error) {
	panic("string register does not have float64 representation")
}

func (r *stringRegister) ReadString(reader Reader) (string, error) {
	data, err := reader.Read(r.baseAddress, r.width, false)
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
	width := uint16(reflect.TypeOf(T(0)).Size() / reflect.TypeOf(uint16(0)).Size())
	length := uint16(1)
	if registerConfig.Length > 1 {
		length = registerConfig.Length
	}
	return &integerRegister{
		register{
			registerConfig.Address,
			width,
		},
		createMappers[T](registerConfig, width),
		length,
		registerConfig.Writable,
	}
}

func createMappers[T uint16 | uint32 | int16 | int32](registerConfig *config.Register, width uint16) mappers {
	inverseFunction := func() func(float64) float64 {
		if inverseFunctionGetter := registerConfig.MapValue.GetInverseFunction; registerConfig.Writable && inverseFunctionGetter != nil {
			inverseFunction, err := inverseFunctionGetter()
			util.PanicOnError(errors.Wrapf(err, "no inverse function for writable register %s", registerConfig.Name))
			return inverseFunction
		}
		return nil
	}()
	mapFromFloat64 := func(value float64) uint16 {
		if validation := registerConfig.Validation; validation != nil {
			util.PanicOnError(errors.Wrapf(validation(value), "validation failed for writable register %s", registerConfig.Name))
		}
		if inverseFunction != nil {
			return uint16(inverseFunction(value))
		}
		return uint16(value)
	}
	return mappers{
		mapToInt64: func(data []uint16) int64 {
			result := T(0)
			for i := uint16(0); i < width; i++ {
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
		mapFromFloat64: mapFromFloat64,
		mapFromString: func(value string) uint16 {
			if mapper := registerConfig.MapValue.ByEnumMap; mapper != nil {
				if mappedValue := util.GetMapKeyForValue(mapper, value); mappedValue != nil {
					return uint16(*mappedValue)
				}
				panic(fmt.Sprintf("cannot find value %s in %v", value, util.GetValues(mapper)))
			}
			floatValue, err := strconv.ParseFloat(value, 64)
			util.PanicOnError(err)
			return mapFromFloat64(floatValue)
		},
	}
}

func (r *integerRegister) getAddressInterval() *util.Interval[uint16] {
	return &util.Interval[uint16]{r.baseAddress, r.baseAddress + (r.length-1)*r.width + (r.width - 1)}
}

func (r *integerRegister) ReadString(reader Reader) (string, error) {
	data, err := reader.Read(r.baseAddress, r.width, r.writable)
	if err != nil {
		return "", err
	}
	return r.mapToString(r.mapToInt64(data)), nil
}

func (r *integerRegister) ReadFloat64(reader Reader, index uint16) (float64, error) {
	if index >= r.length {
		panic("register index out of range")
	}
	data, err := reader.Read(r.baseAddress+index*r.width, r.width, r.writable)
	if err != nil {
		return 0, err
	}
	return r.mapToFloat64(r.mapToInt64(data)), nil
}
