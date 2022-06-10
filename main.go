package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"math"
	"net/http"
	"os"
	configPkg "sungrow-prometheus-exporter/config"
	"sungrow-prometheus-exporter/modbus"
	"sungrow-prometheus-exporter/register"
)

func main() {
	config, err := configPkg.ReadFromFile(getConfigYamlFilename())
	if err != nil {
		panic(err.Error())
	}

	reader := modbus.NewReader(config.Inverter.Address)
	defer reader.Close()

	for _, metricConfig := range config.Metrics {
		registerPrometheusMetric(reader, metricConfig)
	}

	const path = "/metrics"
	http.Handle(path, promhttp.Handler())
	const addr = ":8080"
	log.Infof("Serving at %s%s...", addr, path)
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		panic(err.Error())
	}
}

func getConfigYamlFilename() string {
	dir := "."
	if koDataPath := os.Getenv("KO_DATA_PATH"); len(koDataPath) > 0 {
		dir = koDataPath
	}
	return dir + "/config.yaml"
}

func registerPrometheusMetric(reader register.Reader, metricConfig *configPkg.Metric) {
	labels := prometheus.Labels{}
	for _, labelConfig := range metricConfig.Labels {
		labels[labelConfig.Name] = readStringValue(reader, labelConfig.Value)
	}
	buildPrometheusValueFunc(reader, metricConfig.Value, func(idxValue string, valueFunc func() float64) {
		if len(idxValue) > 0 {
			labels["idx"] = idxValue
		}
		opts := []prometheus.Opts{{
			Namespace:   "sungrow",
			Name:        metricConfig.Name,
			Help:        metricConfig.Help,
			ConstLabels: labels,
		}}
		if len(metricConfig.Alias) > 0 {
			opts = append(opts, prometheus.Opts{
				Name:        metricConfig.Alias,
				Help:        metricConfig.Help,
				ConstLabels: labels,
			})
		}
		for _, opt := range opts {
			if metricConfig.Type == configPkg.Counter {
				promauto.NewCounterFunc(prometheus.CounterOpts(opt), valueFunc)
			}
			if metricConfig.Type == configPkg.Gauge {
				promauto.NewGaugeFunc(prometheus.GaugeOpts(opt), valueFunc)
			}
		}
	})
}

func readStringValue(reader register.Reader, valueConfig *configPkg.Value) string {
	if registerConfig := valueConfig.FromRegister; registerConfig != nil {
		value, err := register.NewFromConfig(registerConfig).ReadString(reader)
		if err != nil {
			panic(err.Error())
		}
		return value
	}
	if expressionConfig := valueConfig.FromExpression; expressionConfig != nil {
		value, err := expressionConfig.Evaluate(map[string]interface{}{})
		if err != nil {
			panic(err.Error())
		}
		return fmt.Sprintf("%v", value)
	}
	panic("cannot read register value for metric")
}

func buildPrometheusValueFunc(reader register.Reader, valueConfig *configPkg.Value, consumer func(idxValue string, valueFunc func() float64)) {
	if registerConfig := valueConfig.FromRegister; registerConfig != nil {
		indexedValueFunc := func(i uint16) float64 {
			value, err := register.NewFromConfig(registerConfig).ReadFloat64(reader, i)
			if err != nil {
				log.Warnf("Cannot read register: %s", err.Error())
				return math.NaN()
			}
			return value
		}
		if registerConfig.Length > 1 {
			for i := uint16(0); i < registerConfig.Length; i++ {
				idx := i
				consumer(fmt.Sprintf("%02d", i), func() float64 {
					return indexedValueFunc(idx)
				})
			}
		} else {
			consumer("", func() float64 {
				return indexedValueFunc(0)
			})
		}
	}
	if expressionConfig := valueConfig.FromExpression; expressionConfig != nil {
		value, err := expressionConfig.Evaluate(map[string]interface{}{})
		if err != nil {
			panic(err.Error())
		}
		consumer("", func() float64 {
			return value.(float64)
		})
	}
}
