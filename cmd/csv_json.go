package cmd

import "foxkit/internal/jsonx"

// jsonIndent: JSON đẹp (2 dấu cách), không escape HTML/ký tự có dấu.
func jsonIndent(v any) ([]byte, error) { return jsonx.Encode(v, "  ") }
