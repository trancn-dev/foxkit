package csvx

import (
	"encoding/json"
	"io"
)

// newEncoder: encoder không escape HTML và không biến ký tự có dấu thành \uXXXX.
func newEncoder(w io.Writer) *json.Encoder {
	e := json.NewEncoder(w)
	e.SetEscapeHTML(false)
	return e
}
