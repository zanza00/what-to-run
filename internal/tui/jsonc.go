// JSONC support: decoding JSON files that allow comments and trailing commas.
package tui

import (
	"encoding/json"

	"github.com/tidwall/jsonc"
)

// unmarshalJSONC decodes JSONC into v: JSON that may additionally contain
// // line comments, /* */ block comments and trailing commas. Standard JSON
// is valid JSONC, so this also decodes a plain .json file.
func unmarshalJSONC(data []byte, v any) error {
	return json.Unmarshal(jsonc.ToJSON(data), v)
}
