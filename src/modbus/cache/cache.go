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
	errs             []error
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
		errs:             make([]error, len(addressIntervals)),
	}
}

func (c *Cache) Read(address uint16, quantity uint16, reader Reader) ([]uint16, error) {
	intervalIdx := c.findInterval(address, quantity)
	if intervalIdx < 0 {
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
			c.update(reader)
		}
	} else {
		defer c.mutex.RUnlock()
	}
	if err := c.errs[intervalIdx]; err != nil {
		return nil, err
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

func (c *Cache) findInterval(address uint16, quantity uint16) int {
	endAddress := address + quantity - 1
	for i, addressInterval := range c.addressIntervals {
		if addressInterval.Contains(address) && addressInterval.End >= endAddress {
			return i
		}
	}
	return -1
}

func (c *Cache) readCache(address uint16, quantity uint16) []uint16 {
	startIdx := address - c.addressIntervals[0].Start
	return c.values[startIdx : startIdx+quantity]
}

func (c *Cache) expired() bool {
	return c.lastUpdate.Before(time.Now().Add(-c.expiry))
}

// update reads each interval on its own, so a range the device rejects
// does not hide the others.
func (c *Cache) update(reader Reader) {
	startAddress := c.addressIntervals[0].Start
	for i, addressInterval := range c.addressIntervals {
		data, err := reader(addressInterval.Start, addressInterval.Length())
		c.errs[i] = err
		if err != nil {
			log.Warnf("Cannot read address range %v: %s", addressInterval, err)
			continue
		}
		copy(c.values[addressInterval.Start-startAddress:], data)
	}
	c.lastUpdate = time.Now()
}
