package modbus

import (
	"fmt"
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

const (
	maxQuantity         = 125
	maxRetries          = 10
	initialRetryBackoff = 30 * time.Millisecond
)

type RegisterReadWriter struct {
	handler    *modbus.TCPClientHandler
	client     modbus.Client
	readCache  *cache.Cache
	writeCache *cache.Cache
}

func NewReadWriter(address string, readAddressIntervals, writeAddressIntervals util.Intervals[uint16]) *RegisterReadWriter {
	handler := modbus.NewTCPClientHandler(address)
	handler.Timeout = 3 * time.Second
	handler.IdleTimeout = 5 * time.Second
	handler.SlaveId = 0x1
	client := modbus.NewClient(handler)
	return &RegisterReadWriter{handler, client,
		cache.New(readAddressIntervals),
		cache.New(writeAddressIntervals),
	}
}

func (r *RegisterReadWriter) Close() {
	err := r.handler.Close()
	util.PanicOnError(err)
}

func (r *RegisterReadWriter) Read(address, quantity uint16, writable bool) ([]uint16, error) {
	c := func() *cache.Cache {
		if writable {
			return r.writeCache
		} else {
			return r.readCache
		}
	}()
	return c.Read(address, quantity, func(address, quantity uint16) ([]uint16, error) {
		return r.readChunked(address, quantity, writable)
	})
}

func (r *RegisterReadWriter) WriteAndReadBack(address uint16, values []uint16) ([]uint16, error) {
	quantity := uint16(len(values))
	log.Infof("Writing address range %d:%d with values %v", address, address+quantity-1, values)
	_, err := r.writeWithRetry(address, quantity, convertUInt16ToBytes(values))
	if err != nil {
		return nil, err
	}
	return r.readChunked(address, quantity, true)
}

func (r *RegisterReadWriter) readChunked(address, quantity uint16, writable bool) ([]uint16, error) {
	var result []byte
	leftToRead := quantity
	offset := uint16(0)
	for leftToRead > 0 {
		chunk, err := r.readWithRetry(address+offset, util.Min(leftToRead, maxQuantity), writable)
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

func (r *RegisterReadWriter) onRetryError() error {
	// always close handler to force reconnect on next read/write
	return errors.Wrapf(r.handler.Close(), "cannot close handler after retry error")
}

func (r *RegisterReadWriter) writeWithRetry(address, quantity uint16, values []byte) (any, error) {
	return doWithRetry(
		fmt.Sprintf("write %d[%d]", address, quantity),
		r.onRetryError,
		func() (any, error) {
			return r.client.WriteMultipleRegisters(address-1, quantity, values)
		},
	)
}

func (r *RegisterReadWriter) readWithRetry(address, quantity uint16, writable bool) ([]byte, error) {
	return doWithRetry(
		fmt.Sprintf("read %d[%d]", address, quantity),
		r.onRetryError,
		func() ([]byte, error) {
			if writable {
				return r.client.ReadHoldingRegisters(address-1, quantity)
			} else {
				return r.client.ReadInputRegisters(address-1, quantity)
			}
		},
	)
}

type retry[R any] struct {
	description string
	onError     func() error
	command     func() (R, error)
}

func doWithRetry[R any](description string, onError func() error, command func() (R, error)) (R, error) {
	return retry[R]{description, onError, command}.doWithRetry(maxRetries, initialRetryBackoff)
}

func (r retry[R]) doWithRetry(retriesLeft int, backoff time.Duration) (R, error) {
	result, err := r.command()
	if err != nil {
		if err := r.onError(); err != nil {
			return result, err
		}
		// Sungrow inverters have the nasty property to RST the TCP connection whenever
		// someone else communicates with the device
		if util.IsAnyError(err, syscall.EPIPE, syscall.ECONNRESET, io.EOF, io.ErrUnexpectedEOF) || os.IsTimeout(err) {
			if retriesLeft == 0 {
				return result, errors.Wrapf(err, "retries exhausted")
			}
			retriesLeft--
			log.Infof("Re-trying %s in %s, %d retries left", r.description, backoff, retriesLeft)
			time.Sleep(backoff)
			return r.doWithRetry(retriesLeft, 2*backoff)
		}
	}
	return result, err
}

func convertBytesToUInt16(bytes []byte) []uint16 {
	// TODO maybe use binary.Read?
	size := len(bytes) / 2
	result := make([]uint16, size)
	for i := 0; i < size; i++ {
		result[i] = uint16(bytes[2*i+1]) + uint16(bytes[2*i])<<8
	}
	return result
}

func convertUInt16ToBytes(data []uint16) []byte {
	// TODO maybe use binary.Read?
	size := len(data)
	result := make([]byte, 2*size)
	for i := 0; i < size; i++ {
		result[2*i] = byte(data[i] >> 8)
		result[2*i+1] = byte(data[i])
	}
	return result
}
