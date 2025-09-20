package utility

import (
	"encoding/json"
)

// UnmarshalJSONAs unmarshals a JSON byte slice into a value of type T
// and returns a pointer to it.
//
// Example:
//
//	user, err := UnmarshalJSONAs[User](jsonData)
func UnmarshalJSONAs[T any](data []byte) (*T, error) {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
