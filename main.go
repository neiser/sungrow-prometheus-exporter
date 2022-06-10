package main

import (
	"github.com/goburrow/modbus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
	configPkg "sungrow-prometheus-exporter/config"
	"sungrow-prometheus-exporter/register"
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
		if registerConfig := metric.Value.FromRegister; registerConfig != nil {
			value, err := register.NewFromConfig(registerConfig).ReadWith(func(address, quantity uint16) ([]byte, error) {
				return client.ReadInputRegisters(address-1, quantity)
			})
			if err != nil {
				panic(err.Error())
			}
			log.Infof("%s=%f", metric.Name, value.AsFloat64())
		}

	}

}
