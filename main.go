package main

import (
	"os"
	configPkg "sungrow-prometheus-exporter/config"
	"sungrow-prometheus-exporter/modbus"
	"sungrow-prometheus-exporter/prometheus"
)

func main() {
	config, err := configPkg.ReadFromFile(getConfigYamlFilename())
	if err != nil {
		panic(err.Error())
	}

	reader := modbus.NewReader(config.Inverter.Address)
	defer reader.Close()

	for _, metricConfig := range config.Metrics {
		prometheus.RegisterMetric(reader, metricConfig)
	}
	prometheus.ListenAndServe()
}

func getConfigYamlFilename() string {
	dir := "."
	if koDataPath := os.Getenv("KO_DATA_PATH"); len(koDataPath) > 0 {
		dir = koDataPath
	}
	return dir + "/config.yaml"
}
