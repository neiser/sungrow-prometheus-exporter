package modbus

import (
	"github.com/goburrow/modbus"
	"time"
)

type RegisterReader struct {
	handler *modbus.TCPClientHandler
	client  modbus.Client
}

func NewReader(address string) (*RegisterReader, error) {
	handler := modbus.NewTCPClientHandler(address)
	handler.Timeout = 3 * time.Second
	handler.SlaveId = 0x1
	err := handler.Connect()
	if err != nil {
		return nil, err
	}
	client := modbus.NewClient(handler)
	return &RegisterReader{handler, client}, nil
}

func (r *RegisterReader) Close() {
	err := r.handler.Close()
	if err != nil {
		panic(err.Error())
	}
}

func (r *RegisterReader) Read(address, quantity uint16) ([]uint16, error) {
	bytes, err := r.client.ReadInputRegisters(address-1, quantity)
	if err != nil {
		return nil, err
	}
	return convertBytesToUInt16(bytes), nil
}

func convertBytesToUInt16(bytes []byte) []uint16 {
	size := len(bytes) / 2
	result := make([]uint16, size)
	for i := 0; i < size; i++ {
		result[i] = uint16(bytes[2*i+1]) + uint16(bytes[2*i])<<8
	}
	return result
}
