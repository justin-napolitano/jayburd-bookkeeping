// Package migrations embeds the reviewed immutable SQL and checksum manifest in the migrator image.
package migrations

import "embed"

// Files contains only versioned SQL and its manifest, never operator configuration.
//
//go:embed *.sql SHA256SUMS
var Files embed.FS
