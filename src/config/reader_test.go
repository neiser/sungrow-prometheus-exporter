package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sungrow-prometheus-exporter/config"
)

func TestReadEmbeddedConfig(t *testing.T) {
	c, err := Read(config.Files)
	require.NoError(t, err)
	assert.NotEmpty(t, c.Metrics)
	for _, name := range c.Metrics.FindRegisterNames() {
		assert.Contains(t, c.Registers, name)
	}
	for _, actuator := range c.Actuators {
		for name := range actuator.Registers {
			assert.Contains(t, c.Registers, name)
		}
	}
}
