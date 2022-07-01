package actuator

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
	"sungrow-prometheus-exporter/src/config"
	"sungrow-prometheus-exporter/src/register"
	"sungrow-prometheus-exporter/src/util"
)

const (
	contentTypeTextPlain = "text/plain"
)

type httpWriter func(text string)
type handleFunc func(writer httpWriter, body string)
type handler struct {
	handleFunc
	match func(r *http.Request) bool
}

func RegisterHttpHandler(basePath string, readWriter register.ReadWriter, actuatorsConfig config.Actuators, registersConfig config.Registers) {
	log.Infof("Serving %d actuators at path %s", len(actuatorsConfig), basePath)
	for actuatorName, actuatorConfig := range actuatorsConfig {
		actuatorConfig := actuatorConfig // prevent stupid capture by reference
		registerHandlers(path.Join(basePath, actuatorName),
			matchPost(func(writer httpWriter, body string) {
				writeValue(writer, actuatorConfig, body, readWriter, registersConfig)
			}),
			matchGet(func(writer httpWriter) {
				readValue(writer, actuatorConfig, readWriter, registersConfig)
			}),
		)
	}
	registerHandlers(basePath, matchGet(func(writer httpWriter) {
		writer(strings.Join(util.GetKeys(actuatorsConfig), "\n"))
	}))
}

func matchPost(handleFunc handleFunc) *handler {
	return matchHttpMethod(http.MethodPost, handleFunc)
}

func matchGet(handleFunc func(writer httpWriter)) *handler {
	return matchHttpMethod(http.MethodGet, func(writer httpWriter, body string) {
		handleFunc(writer)
	})
}

func matchHttpMethod(method string, handleFunc handleFunc) *handler {
	return &handler{handleFunc, func(r *http.Request) bool {
		return r.Method == method
	}}
}

func registerHandlers(path string, handlers ...*handler) {
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				errMessage := fmt.Sprintf("%v", err)
				log.Error(errMessage)
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte(errMessage))
				util.PanicOnError(err)
			}
		}()
		for _, h := range handlers {
			if h.match(r) {
				var body []byte
				if r.Body != nil {
					var err error
					body, err = ioutil.ReadAll(r.Body)
					util.PanicOnError(err)
				}
				writer := httpWriter(func(text string) {
					w.Header().Add("Content-Type", contentTypeTextPlain)
					_, err := w.Write([]byte(text))
					util.PanicOnError(err)
				})
				h.handleFunc(writer, string(body))
				return
			}
		}
		panic(fmt.Sprintf("No handler found for request %v on path %s", *r, path))
	})
}

func readValue(writer httpWriter, actuatorConfig *config.Actuator, reader register.Reader, registersConfig config.Registers) {
	if expressionValue := actuatorConfig.ValueFromExpression; expressionValue != nil {
		value, err := expressionValue.Evaluate(func(registerName string) float64 {
			value, err := register.NewFromConfig(registersConfig[registerName]).ReadFloat64(reader, 0)
			util.PanicOnError(err)
			return value
		})
		util.PanicOnError(err)
		writer(fmt.Sprintf("%v", value))
		return
	}
	if len(actuatorConfig.Registers) == 1 {
		registerName, _ := util.GetOnlyMapElement(actuatorConfig.Registers)
		value, err := register.NewFromConfig(registersConfig[registerName]).ReadString(reader)
		util.PanicOnError(err)
		writer(value)
		return
	}
	panic(fmt.Sprintf("cannot read actuator %s", actuatorConfig.Name))
}

func writeValue(httpWriter httpWriter, actuatorConfig *config.Actuator, value string, registerWriter register.Writer, registersConfig config.Registers) {
	registerNames := util.GetKeys(actuatorConfig.Registers)
	registers := register.NewFromConfigs(registersConfig, registerNames...)
	writtenRegisterValues, err := registers.Write(registerWriter, func(registerName string) (string, *float64) {
		if mapValue := actuatorConfig.Registers[registerName]; mapValue.ByFunction != nil {
			return value, util.PointerTo(mapValue.ByFunction(value))
		}
		return value, nil
	})
	util.PanicOnError(err)
	log.Infof("Wrote registers %s", writtenRegisterValues)
	readValue(httpWriter, actuatorConfig, writtenRegisterValues, registersConfig)
}
