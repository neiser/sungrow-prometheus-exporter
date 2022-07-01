package util

import "errors"

func IsAnyError(err error, targets ...error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

func PanicOnError(err error) {
	if err != nil {
		panic(err.Error())
	}
}
