package modbus

import (
	"github.com/goburrow/modbus"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"sungrow-prometheus-exporter/src/modbus/cache"
	"sungrow-prometheus-exporter/src/util"
	"syscall"
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
	var result []byte
	leftToRead := quantity
	offset := uint16(0)
	for leftToRead > 0 {
		chunk, err := r.readWithRetry(address+offset, util.Min(leftToRead, 125), 10, 30*time.Millisecond)
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

func (r *RegisterReader) readWithRetry(address, quantity uint16, retriesLeft int, backoff time.Duration) ([]byte, error) {
	chunk, err := r.client.ReadInputRegisters(address-1, quantity)
	if err != nil {
		// always close handler to force reconnect on next read
		if err := r.handler.Close(); err != nil {
			return nil, errors.Wrapf(err, "cannot close handler after error")
		}
		// Sungrow inverters have the nasty property to RST the TCP connection whenever
		// someone else communicates with the device
		if util.IsAnyError(err, syscall.EPIPE, syscall.ECONNRESET, io.EOF, io.ErrUnexpectedEOF) || os.IsTimeout(err) {
			if retriesLeft == 0 {
				return nil, errors.Wrapf(err, "retries exhausted")
			}
			retriesLeft--
			log.Infof("Re-trying read %d[%d] in %s, %d retries left", address, quantity, backoff, retriesLeft)
			time.Sleep(backoff)
			return r.readWithRetry(address, quantity, retriesLeft, 2*backoff)
		}
		return nil, err
	}
	return chunk, err
}

func convertBytesToUInt16(bytes []byte) []uint16 {
	size := len(bytes) / 2
	result := make([]uint16, size)
	for i := 0; i < size; i++ {
		result[i] = uint16(bytes[2*i+1]) + uint16(bytes[2*i])<<8
	}
	return result
}
