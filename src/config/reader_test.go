package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

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

func TestRejectDuplicateNames(t *testing.T) {
	var metrics Metrics
	err := yaml.Unmarshal([]byte(`
- name: battery_level
  type: gauge
- name: battery_level
  type: counter
`), &metrics)
	assert.ErrorContains(t, err, "duplicate name 'battery_level'")
}
