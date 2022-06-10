package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"math"
	"net/http"
	configPkg "sungrow-prometheus-exporter/config"
	"sungrow-prometheus-exporter/modbus"
	"sungrow-prometheus-exporter/register"
)

func main() {
	config, err := configPkg.ReadFromFile("config.yaml")
	if err != nil {
		panic(err.Error())
	}

	reader, err := modbus.NewReader(config.Inverter.Address)
	if err != nil {
		panic(err.Error())
	}
	defer reader.Close()

	for _, metricConfig := range config.Metrics {
		registerPrometheusMetric(reader, metricConfig)
	}

	http.Handle("/metrics", promhttp.Handler())
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err.Error())
	}
}

func registerPrometheusMetric(reader register.Reader, metricConfig *configPkg.Metric) {
	labels := prometheus.Labels{}
	for _, labelConfig := range metricConfig.Labels {
		labels[labelConfig.Name] = readStringValue(reader, labelConfig.Value)
	}
	valueFunc := buildPrometheusValueFunc(reader, metricConfig.Value)
	if metricConfig.Type == configPkg.Counter {
		promauto.NewCounterFunc(prometheus.CounterOpts{
			Name:        metricConfig.Name,
			Help:        metricConfig.Help,
			ConstLabels: labels,
		}, valueFunc)
	}
	if metricConfig.Type == configPkg.Gauge {
		promauto.NewGaugeFunc(prometheus.GaugeOpts{
			Name:        metricConfig.Name,
			Help:        metricConfig.Help,
			ConstLabels: labels,
		}, valueFunc)
	}
}

func readStringValue(reader register.Reader, valueConfig *configPkg.Value) string {
	if registerConfig := valueConfig.FromRegister; registerConfig != nil {
		value, err := register.NewFromConfig(registerConfig).ReadWith(reader)
		if err != nil {
			panic(err.Error())
		}
		return value.String()
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

func buildPrometheusValueFunc(reader register.Reader, valueConfig *configPkg.Value) func() float64 {
	if registerConfig := valueConfig.FromRegister; registerConfig != nil {
		return func() float64 {
			value, err := register.NewFromConfig(registerConfig).ReadWith(reader)
			if err != nil {
				return math.NaN()
			}
			return value.AsFloat64s()[0]
		}
	}
	if expressionConfig := valueConfig.FromExpression; expressionConfig != nil {
		value, err := expressionConfig.Evaluate(map[string]interface{}{})
		if err != nil {
			panic(err.Error())
		}
		return func() float64 {
			return value.(float64)
		}
	}
	panic("cannot get value func for metric")
}
