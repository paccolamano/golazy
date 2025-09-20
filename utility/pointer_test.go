package utility

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestPtr(t *testing.T) {
	t.Parallel()

	type input struct {
		data any
	}

	tests := []struct {
		name  string
		input input
	}{
		{
			name: "get pointer to int",
			input: input{
				data: 1,
			},
		},
		{
			name: "get pointer to string",
			input: input{
				data: "foo",
			},
		},
		{
			name: "get pointer to struct",
			input: input{
				data: struct {
					Name string
				}{
					Name: "foo",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Ptr(tt.input.data)

			assert.Equal(t, p == nil, false)
			assert.Equal(t, *p, tt.input.data)
		})
	}
}
