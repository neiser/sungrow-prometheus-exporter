package util

type HasKey interface {
	GetKey() string
}

func GetKeys[K comparable, V any](m map[K]V) (keys []K) {
	for k := range m {
		keys = append(keys, k)
	}
	return
}

func GetValues[K comparable, V any](m map[K]V) (values []V) {
	for _, v := range m {
		values = append(values, v)
	}
	return
}

func MapFromNamedSlice[K HasKey, R any](mapValue func(n K) R, ns ...K) map[string]R {
	r := make(map[string]R)
	for _, n := range ns {
		r[n.GetKey()] = mapValue(n)
	}
	return r
}

func GetOnlyMapElement[K comparable, V any](m map[K]V) (k K, v V) {
	for k, v = range m {
		break
	}
	return
}

func GetMapKeyForValue[K comparable, V comparable](m map[K]V, needle V) *K {
	for k, v := range m {
		if v == needle {
			return &k
		}
	}
	return nil
}
