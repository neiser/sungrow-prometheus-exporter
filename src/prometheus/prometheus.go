package prometheus

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"math"
	"net/http"
	"sungrow-prometheus-exporter/src/config"
	"sungrow-prometheus-exporter/src/register"
	"sungrow-prometheus-exporter/src/util"
)

const namespace = "sungrow"

func RegisterHttpHandler(path string) {
	log.Infof("Serving metrics at path %s", path)
	http.Handle(path, promhttp.Handler())
}

func RegisterMetric(reader register.Reader, metricConfig *config.Metric, registersConfig config.Registers) {
	labels := prometheus.Labels{}
	for _, labelConfig := range metricConfig.Labels {
		labels[labelConfig.Name] = readStringValue(reader, labelConfig.Value, registersConfig)
	}
	buildValueFunc(reader, metricConfig.Value, registersConfig, func(idxValue string, unit string, valueFunc func() float64) {
		if len(idxValue) > 0 {
			labels["idx"] = idxValue
		}
		opts := []prometheus.Opts{{
			Namespace:   namespace,
			Name:        appendPluralUnitToName(metricConfig.Name, unit),
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
			if metricConfig.Type == config.Counter {
				promauto.NewCounterFunc(prometheus.CounterOpts(opt), valueFunc)
			}
			if metricConfig.Type == config.Gauge {
				promauto.NewGaugeFunc(prometheus.GaugeOpts(opt), valueFunc)
			}
		}
	})
}

func appendPluralUnitToName(name string, unit string) string {
	if len(unit) == 0 {
		return name
	}
	if lastCharacter := unit[len(unit)-1]; lastCharacter == 'z' || lastCharacter == 's' {
		return name + "_" + unit
	}
	return name + "_" + unit + "s"
}

func readStringValue(reader register.Reader, valueConfig *config.Value, registersConfig config.Registers) string {
	if registerValue := valueConfig.FromRegister; registerValue != nil {
		registerConfig := registersConfig[registerValue.Name]
		value, err := register.NewFromConfig(registerConfig).ReadString(reader)
		if err != nil {
			panic(err.Error())
		}
		return value
	}
	if expressionConfig := valueConfig.FromExpression; expressionConfig != nil {
		value, err := expressionConfig.Evaluate(func(registerName string) float64 {
			return readRegister(registersConfig[registerName], reader, 0)
		})
		if err != nil {
			panic(err.Error())
		}
		return fmt.Sprintf("%v", value)
	}
	panic("cannot read register value for metric")
}

func buildValueFunc(reader register.Reader, valueConfig *config.Value, registersConfig config.Registers, consumer func(idxValue string, unit string, valueFunc func() float64)) {
	if registerValue := valueConfig.FromRegister; registerValue != nil {
		registerConfig := registersConfig[registerValue.Name]
		if registerConfig.Length > 1 {
			for i := uint16(0); i < registerConfig.Length; i++ {
				index := i // prevent lambda capture by reference!
				consumer(fmt.Sprintf("%02d", index), registerConfig.Unit, func() float64 {
					return readRegister(registerConfig, reader, index)
				})
			}
		} else {
			consumer("", registerConfig.Unit, func() float64 {
				return readRegister(registerConfig, reader, 0)
			})
		}
	}
	if expressionConfig := valueConfig.FromExpression; expressionConfig != nil {
		consumer("", "", func() float64 {
			value, err := expressionConfig.Evaluate(func(registerName string) float64 {
				return readRegister(registersConfig[registerName], reader, 0)
			})
			if err != nil {
				panic(err.Error())
			}
			return util.NumericToFloat64(value)
		})
	}
}

func readRegister(registerConfig *config.Register, reader register.Reader, index uint16) float64 {
	value, err := register.NewFromConfig(registerConfig).ReadFloat64(reader, index)
	if err != nil {
		log.Warnf("Cannot read register: %s", err.Error())
		return math.NaN()
	}
	return value
}
