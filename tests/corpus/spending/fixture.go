// Package spending exposes original fictional inputs to disposable proof binaries.
// Application packages do not import this package; it carries no private data.
package spending

import _ "embed"

// Observations is hand-authored source evidence, independent of runtime aggregation.
//
//go:embed observations.json
var Observations []byte
