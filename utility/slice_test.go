package utility

import (
	"strconv"
	"testing"

	"gotest.tools/v3/assert"
)

func TestMap(t *testing.T) {
	t.Parallel()

	numbers := []int{1, 2, 3, 4, 5}
	stringNumbers := Map(numbers, func(num int) string {
		return strconv.Itoa(num)
	})

	assert.DeepEqual(t, []string{"1", "2", "3", "4", "5"}, stringNumbers)
}

func TestMapE(t *testing.T) {
	t.Parallel()

	type input struct {
		data []string
	}

	type output struct {
		res         []int
		expectedErr string
	}

	tests := []struct {
		name   string
		input  input
		output output
	}{
		{
			name: "mapE",
			input: input{
				data: []string{"1", "2", "3", "4", "5"},
			},
			output: output{
				res: []int{1, 2, 3, 4, 5},
			},
		},
		{
			name: "map should fail",
			input: input{
				data: []string{"1", "2", "3", "4", "NaN"},
			},
			output: output{
				expectedErr: `parsing "NaN": invalid syntax`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			numbers, err := MapE(tt.input.data, func(letter string) (int, error) {
				return strconv.Atoi(letter)
			})
			if tt.output.expectedErr != "" {
				assert.ErrorContains(t, err, tt.output.expectedErr)
			} else {
				assert.NilError(t, err)
				assert.DeepEqual(t, numbers, tt.output.res)
			}
		})
	}
}
