package main

import (
	"github.com/goburrow/modbus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
	configPkg "sungrow-prometheus-exporter/config"
	"time"
)

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myapp_processed_ops_total",
		Help: "The total number of processed events",
	})
)

func main() {
	config, err := configPkg.ReadFromFile("config.yaml")
	if err != nil {
		panic(err.Error())
	}

	handler := modbus.NewTCPClientHandler(config.Inverter.Address)
	handler.Timeout = 3 * time.Second
	handler.SlaveId = 0x1
	// Connect manually so that multiple requests are handled in one connection session
	err = handler.Connect()
	if err != nil {
		panic(err.Error())
	}
	defer func(handler *modbus.TCPClientHandler) {
		err := handler.Close()
		if err != nil {
			panic(err.Error())
		}
	}(handler)

	client := modbus.NewClient(handler)

	for _, metric := range config.Metrics {
		if registerValue, ok := metric.Value().(*configPkg.RegisterValue); ok {
			if registerValue.Type == configPkg.U16RegisterType {
				results, err := client.ReadInputRegisters(registerValue.Address-1, 1)
				if err != nil {
					panic(err.Error())
				}
				log.Infof("%s=%f", metric.Name, float64(uint16(results[1])+256*uint16(results[0]))/10)
			}
		}

	}

}
