package genjson

import (
	"encoding/json"

	"github.com/Invicton-Labs/go-stackerr"
)

func Unmarshal[T any](data []byte) (v T, err stackerr.Error) {
	if err := json.Unmarshal(data, &v); err != nil {
		return v, stackerr.Wrap(err)
	}
	return v, err
}
