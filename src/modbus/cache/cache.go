package cache

import (
	log "github.com/sirupsen/logrus"
	"sungrow-prometheus-exporter/src/util"
	"sync"
	"time"
)

type Reader func(address, quantity uint16) ([]uint16, error)

type Cache struct {
	expiry           time.Duration
	addressIntervals util.Intervals[uint16]
	values           []uint16
	lastUpdate       time.Time
	mutex            sync.RWMutex
}

func New(addressIntervals util.Intervals[uint16]) *Cache {
	if len(addressIntervals) == 0 {
		return &Cache{}
	}
	addressIntervals.SortAndMerge()
	log.Infof("Initializing cache for address intervals %v", addressIntervals)
	return &Cache{
		expiry:           500 * time.Millisecond,
		addressIntervals: addressIntervals,
		values:           make([]uint16, getSize(addressIntervals)),
	}
}

func (c *Cache) Read(address uint16, quantity uint16, reader Reader) ([]uint16, error) {
	if c.isAddressOutsideCache(address, quantity) {
		return reader(address, quantity)
	}
	c.mutex.RLock()
	if c.expired() {
		// upgrade to rw lock
		c.mutex.RUnlock()
		c.mutex.Lock()
		defer c.mutex.Unlock()
		// check expired again within rw lock
		// to see if other thread has updated cache in the meantime
		if c.expired() {
			err := c.update(reader)
			if err != nil {
				return nil, err
			}
		}
	} else {
		defer c.mutex.RUnlock()
	}
	return c.readCache(address, quantity), nil
}

func getSize(addressIntervals util.Intervals[uint16]) uint16 {
	startAddress := addressIntervals[0].Start
	endAddress := addressIntervals[len(addressIntervals)-1].End
	for _, addressInterval := range addressIntervals {
		if endAddress < addressInterval.End {
			endAddress = addressInterval.End
		}
	}
	return endAddress - startAddress + 1
}

func (c *Cache) isAddressOutsideCache(address uint16, quantity uint16) bool {
	endAddress := address + quantity - 1
	for _, addressInterval := range c.addressIntervals {
		if addressInterval.Contains(address) && addressInterval.End >= endAddress {
			return false
		}
	}
	return true
}

func (c *Cache) readCache(address uint16, quantity uint16) []uint16 {
	startIdx := address - c.addressIntervals[0].Start
	return c.values[startIdx : startIdx+quantity]
}

func (c *Cache) expired() bool {
	return c.lastUpdate.Before(time.Now().Add(-c.expiry))
}

func (c *Cache) update(reader Reader) error {
	startAddress := c.addressIntervals[0].Start
	for _, addressInterval := range c.addressIntervals {
		quantity := addressInterval.Length()
		data, err := reader(addressInterval.Start, quantity)
		if err != nil {
			return err
		}
		startIdx := addressInterval.Start - startAddress
		for i := uint16(0); i < quantity; i++ {
			c.values[startIdx+i] = data[i]
		}
	}
	c.lastUpdate = time.Now()
	return nil
}
