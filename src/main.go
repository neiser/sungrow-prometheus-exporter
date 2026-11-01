package main

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	embeddedConfig "sungrow-prometheus-exporter/config"
	"sungrow-prometheus-exporter/src/actuator"
	configPkg "sungrow-prometheus-exporter/src/config"
	"sungrow-prometheus-exporter/src/modbus"
	"sungrow-prometheus-exporter/src/prometheus"
	"sungrow-prometheus-exporter/src/register"
	"sungrow-prometheus-exporter/src/util"
)

func main() {

	var inverterAddress, listenAddress, configDir string

	rootCmd := &cobra.Command{
		Use:   "sungrow-prometheus-exporter",
		Short: "Prometheus Exporter for Sungrow inverters",
		RunE: func(cmd *cobra.Command, args []string) error {
			var configFiles fs.FS = embeddedConfig.Files
			if configDir != "" {
				configFiles = os.DirFS(configDir)
			}
			config, err := configPkg.Read(configFiles)
			if err != nil {
				return err
			}

			readAddressIntervals, writeAddressIntervals := register.FindAddressIntervals(config.Registers,
				config.Metrics.FindRegisterNames()...,
			)
			readWriter := modbus.NewReadWriter(inverterAddress, readAddressIntervals, writeAddressIntervals)
			defer readWriter.Close()

			for _, metricConfig := range config.Metrics {
				prometheus.RegisterMetric(readWriter, metricConfig, config.Registers)
			}
			prometheus.RegisterHttpHandler("/")

			actuator.RegisterHttpHandler("/actuator", readWriter, config.Actuators, config.Registers)

			listenAndServe(listenAddress)
			return nil
		},
	}

	rootCmd.Flags().StringVar(&inverterAddress, "inverter-address", "sungrow:502", "Address as 'host:port' of inverter")
	rootCmd.Flags().StringVar(&listenAddress, "listen-address", ":8080", "Address as 'host:port' to serve metrics and actuators on")
	rootCmd.Flags().StringVar(&configDir, "config-dir", "", "Directory with metrics.yaml, registers.yaml and actuators.yaml (default: the embedded files)")

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func listenAndServe(address string) {
	log.Infof("Listening at %s", address)
	err := http.ListenAndServe(address, nil)
	util.PanicOnError(err)
}
