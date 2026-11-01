package cache

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sungrow-prometheus-exporter/src/util"
)

func TestReadKeepsWorkingIntervalsWhenOneFails(t *testing.T) {
	errIllegalAddress := errors.New("illegal data address")
	var reads []util.Interval[uint16]
	reader := func(address, quantity uint16) ([]uint16, error) {
		reads = append(reads, util.Interval[uint16]{Start: address, End: address + quantity - 1})
		if address == 6100 {
			return nil, errIllegalAddress
		}
		values := make([]uint16, quantity)
		for i := range values {
			values[i] = address + uint16(i)
		}
		return values, nil
	}
	c := New(util.Intervals[uint16]{
		{Start: 5000, End: 5003},
		{Start: 6100, End: 6195},
		{Start: 13000, End: 13001},
	})

	values, err := c.Read(5001, 2, reader)
	require.NoError(t, err)
	assert.Equal(t, []uint16{5001, 5002}, values)
	assert.Len(t, reads, 3)

	_, err = c.Read(6150, 1, reader)
	assert.ErrorIs(t, err, errIllegalAddress)

	values, err = c.Read(13000, 2, reader)
	require.NoError(t, err)
	assert.Equal(t, []uint16{13000, 13001}, values)
	assert.Len(t, reads, 3, "values within the expiry come from the cache")

	values, err = c.Read(7000, 1, reader)
	require.NoError(t, err)
	assert.Equal(t, []uint16{7000}, values)
}
