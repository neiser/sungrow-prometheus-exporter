package actuator

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"path"
)

func RegisterHttpHandler(basePath string) {
	log.Infof("Serving actuator at path %s/", basePath)
	http.HandleFunc(basePath+"/", func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodPost:
			handlePost(writer, request)
		default:
			panic(fmt.Sprintf("Unsupported HTTP method %s", request.Method))
		}
	})
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	b := bytes.Buffer{}
	_, err := io.Copy(&b, r.Body)
	if err != nil {
		panic(err)
	}
	actuatorName := path.Base(r.URL.Path)
	actuatorValue := b.String()
	_, err = w.Write([]byte(fmt.Sprintf("Hello World POST %s=%s ", actuatorName, actuatorValue)))
	if err != nil {
		panic(err)
	}
}
