package util

func MapSlice[T any, R any](ts []T, mapper func(T) R) (rs []R) {
	for _, t := range ts {
		rs = append(rs, mapper(t))
	}
	return
}
