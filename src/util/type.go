package util

func CastToBool(v interface{}) bool {
	return v.(bool)
}

func PointerTo[T any](v T) *T {
	return &v
}
