package utility

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestUnmarshalJSONAs(t *testing.T) {
	t.Parallel()

	type Person struct {
		Name string `json:"name"`
	}

	type input struct {
		rawJSON []byte
	}

	type output[T any] struct {
		expectedRes *T
		expectedErr string
	}

	tests := []struct {
		name   string
		input  input
		output output[Person]
	}{
		{
			name: "should unmarshal struct",
			input: input{
				rawJSON: []byte(`{"name":"foo"}`),
			},
			output: output[Person]{
				expectedRes: &Person{
					Name: "foo",
				},
			},
		},
		{
			name: "unmarshal struct should fail",
			input: input{
				rawJSON: []byte(`{"name":123}`),
			},
			output: output[Person]{
				expectedRes: nil,
				expectedErr: "json: cannot unmarshal number",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := UnmarshalJSONAs[Person](tt.input.rawJSON)
			if tt.output.expectedErr != "" {
				assert.ErrorContains(t, err, tt.output.expectedErr)
			} else {
				assert.NilError(t, err)
			}

			assert.DeepEqual(t, res, tt.output.expectedRes)
		})
	}
}
