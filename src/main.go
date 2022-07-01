package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"sungrow-prometheus-exporter/src/actuator"
	configPkg "sungrow-prometheus-exporter/src/config"
	"sungrow-prometheus-exporter/src/modbus"
	"sungrow-prometheus-exporter/src/prometheus"
	"sungrow-prometheus-exporter/src/register"
	"sungrow-prometheus-exporter/src/util"
)

func main() {

	var inverterAddress string

	rootCmd := &cobra.Command{
		Use:   "sungrow-prometheus-exporter",
		Short: "Prometheus Exporter for Sungrow inverters",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := configPkg.Read()
			if err != nil {
				return err
			}

			readAddressIntervals, writeAddressIntervals := register.FindAddressIntervals(config.Registers,
				config.Metrics.FindRegisterNames()...,
			)
			reader := modbus.NewReader(inverterAddress, readAddressIntervals, writeAddressIntervals)
			defer reader.Close()

			for _, metricConfig := range config.Metrics {
				prometheus.RegisterMetric(reader.Read, metricConfig, config.Registers)
			}
			prometheus.RegisterHttpHandler("/")

			actuator.RegisterHttpHandler("/actuator", config.Actuators, config.Registers)

			listenAndServe(8080)
			return nil
		},
	}

	rootCmd.Flags().StringVar(&inverterAddress, "inverter-address", "sungrow:502", "Address as 'host:port' of inverter")

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func listenAndServe(port int) {
	address := fmt.Sprintf(":%d", port)
	log.Infof("Listening at %s", address)
	err := http.ListenAndServe(address, nil)
	util.PanicOnError(err)
}
