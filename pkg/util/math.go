package util

// Numeric is a constraint for numeric types that support comparison and arithmetic.
type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// Avg calculates the average of a slice of numeric values.
// Returns the zero value if the slice is empty.
func Avg[T Numeric](values []T) T {
	if len(values) == 0 {
		return 0
	}
	var sum T
	for _, v := range values {
		sum += v
	}
	return sum / T(len(values))
}

// Min returns the minimum value in a slice.
// Returns the zero value if the slice is empty.
func Min[T Numeric](values []T) T {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// Max returns the maximum value in a slice.
// Returns the zero value if the slice is empty.
func Max[T Numeric](values []T) T {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
