package util

func GetOnlyMapElement[K comparable, V any](m map[K]V) (k K, v V) {
	for k, v = range m {
		break
	}
	return
}
