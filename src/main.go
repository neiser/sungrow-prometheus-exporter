package main

import (
	"os"
	configPkg "sungrow-prometheus-exporter/src/config"
	"sungrow-prometheus-exporter/src/modbus"
	"sungrow-prometheus-exporter/src/prometheus"
	"sungrow-prometheus-exporter/src/register"
)

func main() {
	config, err := configPkg.ReadFromFile(getConfigYamlFilename())
	if err != nil {
		panic(err.Error())
	}

	addressIntervals := register.FindAddressIntervals(config.Metrics.FindRegisters())
	reader := modbus.NewReader(config.Inverter.Address, addressIntervals)
	defer reader.Close()

	for _, metricConfig := range config.Metrics {
		prometheus.RegisterMetric(reader.Read, metricConfig)
	}
	prometheus.ListenAndServe("/", 8080)
}

func getConfigYamlFilename() string {
	dir := "."
	if koDataPath := os.Getenv("KO_DATA_PATH"); len(koDataPath) > 0 {
		dir = koDataPath
	}
	return dir + "/config.yaml"
}
