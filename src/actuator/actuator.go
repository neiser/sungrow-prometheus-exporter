package actuator

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"sungrow-prometheus-exporter/src/config"
	"sungrow-prometheus-exporter/src/register"
	"sungrow-prometheus-exporter/src/util"
)

const (
	contentTypeTextPlain = "text/plain"
)

func RegisterHttpHandler(basePath string, actuatorsConfig config.Actuators, registersConfig config.Registers) {
	log.Infof("Serving actuator at path %s/", basePath)
	http.HandleFunc(basePath+"/", func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodPost:
			handlePost(writer, request, actuatorsConfig, registersConfig)
		default:
			panic(fmt.Sprintf("Unsupported HTTP method %s", request.Method))
		}
	})
}

func handlePost(w http.ResponseWriter, r *http.Request, actuatorsConfig config.Actuators, registersConfig config.Registers) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
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
		log.Infof("Would write %d to %d with values %v", address, address+quantity-1, values)
		return values, nil
	}, func(registerName string) (uint16, error) {
		registerConfig := registersConfig[registerName]

		// TODO validation, and consider numeric input (as string)

		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {

		}

		mapValue := actuatorConfig.Registers[registerName]
		if mapValue.ByFunction != nil {
			return mapValue.ByFunction(value), nil
		}
		if byEnumMap := registerConfig.MapValue.ByEnumMap; len(byEnumMap) > 0 {
			mappedValue := util.GetMapKeyForValue(registerConfig.MapValue.ByEnumMap, value)
			if mappedValue != nil {
				return uint16(*mappedValue), nil
			}
		}

		return 0, fmt.Errorf("cannot find value for register %s and '%s'", registerName, value)
	})
	if err != nil {
		panic(err.Error())
	}
	w.Header().Set("Content-Type", contentTypeTextPlain)
	_, err = w.Write([]byte(fmt.Sprintf("Wrote %v", writtenRegisters)))
	if err != nil {
		panic(err.Error())
	}
}
