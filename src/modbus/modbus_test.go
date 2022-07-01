package modbus

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConvert(t *testing.T) {
	arr1 := []uint16{1, 2, 4000, 1<<16 - 1}
	arr2 := []byte{1<<8 - 1, 0, 1, 5}
	assert.Equal(t, convertBytesToUInt16(convertUInt16ToBytes(arr1)), arr1)
	assert.Equal(t, convertUInt16ToBytes(convertBytesToUInt16(arr2)), arr2)
}
