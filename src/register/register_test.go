package register

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	embeddedConfig "sungrow-prometheus-exporter/config"
	"sungrow-prometheus-exporter/src/config"
)

type fakeReader map[uint16]uint16

func (f fakeReader) Read(address, quantity uint16, _ bool) ([]uint16, error) {
	values := make([]uint16, quantity)
	for i := range values {
		values[i] = f[address+uint16(i)]
	}
	return values, nil
}

func TestReadEmbeddedRegisters(t *testing.T) {
	c, err := config.Read(embeddedConfig.Files)
	require.NoError(t, err)
	// Values read from an SH10RT at night, plus a discharging battery.
	reader := fakeReader{
		5004: 37885, 5005: 8,
		5019: 2327,
		5033: 65532, 5034: 65535,
		5036: 500,
		13008: 505, 13009: 0,
		13010: 65008, 13011: 65535,
		13021: 0xFFE2, // -3.0 A
		13022: 0xFC18, // -1000 W
		13023: 188,
	}
	for name, want := range map[string]float64{
		"R011_total_output_energy": 56217300,
		"R021_phase_a_voltage":     232.7,
		"R025_reactive_power":      -4,
		"R027_grid_frequency":      50,
		"R056_load_power":          505,
		"R057_export_power":        -528,
		"R064_battery_current":     3,
		"R065_battery_power":       1000,
		"R066_battery_level":       0.188,
	} {
		got, err := NewFromConfig(c.Registers[name]).ReadFloat64(reader, 0)
		require.NoError(t, err, name)
		assert.InDelta(t, want, got, 1e-9, name)
	}
}
