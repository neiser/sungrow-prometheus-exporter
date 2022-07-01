package actuator

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"path"
	"sungrow-prometheus-exporter/src/config"
	"sungrow-prometheus-exporter/src/register"
	"sungrow-prometheus-exporter/src/util"
)

const (
	contentTypeTextPlain = "text/plain"
)

func RegisterHttpHandler(basePath string, actuatorsConfig config.Actuators, registersConfig config.Registers) {
	log.Infof("Serving actuator at path %s/", basePath)
	http.HandleFunc(basePath+"/", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte(fmt.Sprintf("%v", err)))
				util.PanicOnError(err)
			}
		}()
		switch r.Method {
		case http.MethodPost:
			handlePost(w, r, actuatorsConfig, registersConfig)
		default:
			panic(fmt.Sprintf("Unsupported HTTP method %s", r.Method))
		}
	})
}

func handlePost(w http.ResponseWriter, r *http.Request, actuatorsConfig config.Actuators, registersConfig config.Registers) {
	body, err := ioutil.ReadAll(r.Body)
	util.PanicOnError(err)
	actuatorName := path.Base(r.URL.Path)
	actuatorValue := string(body)
	if actuatorConfig, ok := actuatorsConfig[actuatorName]; ok {
		writeValue(w, actuatorConfig, actuatorValue, registersConfig)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func writeValue(w http.ResponseWriter, actuatorConfig *config.Actuator, value string, registersConfig config.Registers) {
	registers := register.NewFromConfigs(registersConfig, util.GetKeys(actuatorConfig.Registers)...)
	writtenRegisters, err := registers.Write(func(address, quantity uint16, values []uint16) ([]uint16, error) {
		// TODO use actual writer (from modbus)
		log.Infof("Would write address range [%d:%d] with values %v", address, address+quantity-1, values)
		return values, nil
	}, func(registerName string) (string, *float64) {
		if mapValue := actuatorConfig.Registers[registerName]; mapValue.ByFunction != nil {
			return value, util.PointerTo(mapValue.ByFunction(value))
		}
		return value, nil
	})
	util.PanicOnError(err)
	w.Header().Set("Content-Type", contentTypeTextPlain)
	_, err = w.Write([]byte(fmt.Sprintf("Wrote %v", writtenRegisters)))
	util.PanicOnError(err)
}
