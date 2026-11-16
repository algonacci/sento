package workspace

import _ "embed"

// Schema is the portable workspace contract for editors and AI generators.
//
//go:embed schema.json
var Schema []byte
