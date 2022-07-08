package modbus

import (
	"fmt"
	"github.com/goburrow/modbus"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
	"io"
	"os"
	"sungrow-prometheus-exporter/src/modbus/cache"
	"sungrow-prometheus-exporter/src/util"
	"syscall"
	"time"
)

const (
	maxQuantity = 125

	maxReadWriteRetries          = 10
	initialReadWriteRetryBackoff = 30 * time.Millisecond
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
	return r.awaitStableRead(address, values)
}

func (r *RegisterReadWriter) awaitStableRead(address uint16, expectedValues []uint16) ([]uint16, error) {
	quantity := uint16(len(expectedValues))
	var previouslyReadValues [][]uint16
	var errNotEqual = errors.New("read values not equal to expected values")
	findUnstableIndexes := func() []int {
		var unstableIndexes []int
		for i := 0; i < len(previouslyReadValues)-1; i++ {
			unstableIndexes = append(unstableIndexes, util.FindUnequalIndexes(previouslyReadValues[i], previouslyReadValues[i+1])...)
		}
		if len(unstableIndexes) == 0 {
			return nil
		}
		slices.Sort(unstableIndexes)
		unstableIndexes = slices.Compact(unstableIndexes)
		log.Infof("Found unstable indexes %v from previous reads", unstableIndexes)
		return unstableIndexes
	}
	return retry[[]uint16]{
		description: fmt.Sprintf("stable read %d[%d]", address, quantity),
		onError: func(commandErr error) (bool, error) {
			if commandErr == errNotEqual {
				return true, nil
			}
			return false, errors.Wrap(commandErr, "stopped waiting for stable read")
		},
		command: func() ([]uint16, error) {
			readValues, err := r.readChunked(address, quantity, true)
			log.Infof("Read values %v", readValues)
			if err != nil {
				return nil, err
			}
			unstableIndexes := findUnstableIndexes()
			if slices.CompareFunc(expectedValues, readValues, util.CompareIgnoring[uint16](unstableIndexes)) == 0 {
				return readValues, nil
			}
			previouslyReadValues = append(previouslyReadValues, readValues)
			log.Infof("Awaiting stable read after write, so far %d unstable indexes", len(unstableIndexes))
			return nil, errNotEqual
		},
	}.doWithRetry(6, 100*time.Millisecond)
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

func (r *RegisterReadWriter) onReadWriteRetryError(commandErr error) (bool, error) {
	// always close handler to force reconnect on next read/write
	err := r.handler.Close()
	if err != nil {
		return false, errors.Wrapf(commandErr, "cannot close handler after retry error")
	}
	// Sungrow inverters have the nasty property to RST the TCP connection whenever
	// someone else communicates with the device
	return util.IsAnyError(commandErr, syscall.EPIPE, syscall.ECONNRESET, io.EOF, io.ErrUnexpectedEOF) || os.IsTimeout(commandErr), nil
}

func (r *RegisterReadWriter) writeWithRetry(address, quantity uint16, values []byte) (any, error) {
	return doWithRetry(
		fmt.Sprintf("write %d[%d]", address, quantity),
		r.onReadWriteRetryError,
		func() (any, error) {
			return r.client.WriteMultipleRegisters(address-1, quantity, values)
		},
	)
}

func (r *RegisterReadWriter) readWithRetry(address, quantity uint16, writable bool) ([]byte, error) {
	return doWithRetry(
		fmt.Sprintf("read %d[%d]", address, quantity),
		r.onReadWriteRetryError,
		func() ([]byte, error) {
			if writable {
				return r.client.ReadHoldingRegisters(address-1, quantity)
			} else {
				return r.client.ReadInputRegisters(address-1, quantity)
			}
		},
	)
}

func doWithRetry[R any](description string, onError func(commandErr error) (bool, error), command func() (R, error)) (R, error) {
	return retry[R]{description, onError, command}.doWithRetry(maxReadWriteRetries, initialReadWriteRetryBackoff)
}

type retry[R any] struct {
	description string
	onError     func(commandErr error) (bool, error)
	command     func() (R, error)
}

func (r retry[R]) doWithRetry(retriesLeft int, backoff time.Duration) (R, error) {
	result, commandErr := r.command()
	if commandErr != nil {
		if shouldRetry, err := r.onError(commandErr); err != nil {
			return result, err
		} else if shouldRetry {
			if retriesLeft == 0 {
				return result, errors.Wrapf(commandErr, "retries exhausted")
			}
			retriesLeft--
			log.Infof("Re-trying %s in %s, %d retries left", r.description, backoff, retriesLeft)
			time.Sleep(backoff)
			return r.doWithRetry(retriesLeft, 2*backoff)
		}
	}
	return result, commandErr
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
