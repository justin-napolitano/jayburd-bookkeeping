package migration

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/justin-napolitano/jayburd-bookkeeping/db/migrations"
)

// copyAssets isolates mutations from the reviewed embedded migration corpus.
func copyAssets(t *testing.T) fstest.MapFS {
	t.Helper()
	out := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		b, err := fs.ReadFile(migrations.Files, entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		out[entry.Name()] = &fstest.MapFile{Data: b}
	}
	return out
}

// TestArtifactIntegrity rejects tampering, missing pairs, extra SQL and manifest ambiguity offline.
func TestArtifactIntegrity(t *testing.T) {
	if _, err := Verify(migrations.Files); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"changed", "missing", "extra", "manifest"} {
		t.Run(kind, func(t *testing.T) {
			files := copyAssets(t)
			switch kind {
			case "changed":
				files["000001_ledger.up.sql"] = &fstest.MapFile{Data: []byte("SELECT 1;")}
			case "missing":
				delete(files, "000001_ledger.down.sql")
			case "extra":
				files["999999_unregistered.up.sql"] = &fstest.MapFile{Data: []byte("SELECT 1;")}
			case "manifest":
				files["SHA256SUMS"] = &fstest.MapFile{Data: []byte("malformed")}
			}
			if _, err := Verify(files); err == nil {
				t.Fatal("unsafe artifacts accepted")
			}
		})
	}
}
