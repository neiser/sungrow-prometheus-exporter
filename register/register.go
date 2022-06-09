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

func New(config configPkg.RegisterValue) Register {
	switch config.Type {
	case configPkg.U16RegisterType:
		return &u16Register{
			register{address: config.Address},
		}
	}
	panic("unknown register type")
}

type register struct {
	address uint16
}

type integerMapper func(value uint64) float64

type integerRegister struct {
	mapper integerMapper
}

type u16Register struct {
	register
}

type lazyIntegerValue struct {
	bytes []byte
}

func (l lazyIntegerValue) AsString() string {
	return fmt.Sprintf("%f", l.AsFloat64())
}

func (l lazyIntegerValue) AsFloat64() float64 {
	v := uint64(0)
	for i := len(l.bytes) - 1; i >= 0; i-- {
		v += uint64(l.bytes[i]) * uint64(1<<8*i)
	}
	return float64(v)
}

func (l lazyIntegerValue) asUint64() uint64 {
	v := uint64(0)
	for i := len(l.bytes) - 1; i >= 0; i-- {
		v += uint64(l.bytes[i]) * uint64(1<<8*i)
	}
	return v
}

func (u16 u16Register) ReadWith(reader Reader) (Value, error) {
	bytes, err := reader(u16.address, 1)
	if err != nil {
		return nil, err
	}
	return &lazyIntegerValue{bytes}, nil
}
