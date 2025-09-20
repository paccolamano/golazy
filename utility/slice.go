package utility

// Map returns a new slice of []b from slice []a.
func Map[A any, B any](input []A, f func(A) B) []B {
	output, _ := mapInternal(input, func(a A) (B, error) {
		return f(a), nil
	})
	return output
}

// MapE returns a new slice of []b and error from slice []a.
func MapE[A any, B any](input []A, f func(A) (B, error)) ([]B, error) {
	s, err := mapInternal(input, f)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func mapInternal[A any, B any](input []A, f func(A) (B, error)) ([]B, error) {
	output := make([]B, len(input))
	for i, v := range input {
		mapped, err := f(v)
		if err != nil {
			return nil, err
		}
		output[i] = mapped
	}
	return output, nil
}
