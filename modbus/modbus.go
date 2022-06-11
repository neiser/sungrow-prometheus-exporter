package modbus

import (
	"github.com/goburrow/modbus"
	"sungrow-prometheus-exporter/modbus/cache"
	"sungrow-prometheus-exporter/util"
	"time"
)

type RegisterReader struct {
	handler *modbus.TCPClientHandler
	client  modbus.Client
	cache   *cache.Cache
}

func NewReader(address string, addressIntervals util.Intervals[uint16]) *RegisterReader {
	handler := modbus.NewTCPClientHandler(address)
	handler.Timeout = 3 * time.Second
	handler.IdleTimeout = 5 * time.Second
	handler.SlaveId = 0x1
	client := modbus.NewClient(handler)
	return &RegisterReader{handler, client, cache.New(addressIntervals)}
}

func (r *RegisterReader) Close() {
	err := r.handler.Close()
	if err != nil {
		panic(err.Error())
	}
}

func (r *RegisterReader) Read(address, quantity uint16) ([]uint16, error) {
	return r.cache.Read(address, quantity, r.readChunked)
}

func (r *RegisterReader) readChunked(address, quantity uint16) ([]uint16, error) {
	err := r.handler.Connect()
	if err != nil {
		return nil, err
	}
	var result []byte
	leftToRead := quantity
	offset := uint16(0)
	for leftToRead > 0 {
		chunk, err := r.client.ReadInputRegisters(address-1+offset, util.Min(leftToRead, 125))
		if err != nil {
			return nil, err
		}
		result = append(result, chunk...)
		read := uint16(len(chunk)) / 2
		leftToRead -= read
		offset += read
	}
	return convertBytesToUInt16(result), nil
}

func convertBytesToUInt16(bytes []byte) []uint16 {
	size := len(bytes) / 2
	result := make([]uint16, size)
	for i := 0; i < size; i++ {
		result[i] = uint16(bytes[2*i+1]) + uint16(bytes[2*i])<<8
	}
	return result
}
