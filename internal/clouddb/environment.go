package clouddb

// forbiddenEnvironment checks presence only, preventing SDK emulators, metadata
// redirection, credential-file discovery and inherited PostgreSQL overrides.
func ForbiddenEnvironment(present func(string) bool) bool {
	for _, name := range []string{"FIREBASE_AUTH_EMULATOR_HOST", "FIREBASE_CONFIG", "GOOGLE_APPLICATION_CREDENTIALS", "GCE_METADATA_HOST", "GCE_METADATA_IP", "PGHOST", "PGHOSTADDR", "PGPORT", "PGDATABASE", "PGUSER", "PGPASSWORD", "PGPASSFILE", "PGSERVICE", "PGSERVICEFILE", "PGSSLMODE", "PGSSLCERT", "PGSSLKEY", "PGSSLROOTCERT", "PGSSLPASSWORD", "PGOPTIONS", "PGAPPNAME", "PGCONNECT_TIMEOUT", "PGTARGETSESSIONATTRS", "PGTZ", "PGMINPROTOCOLVERSION", "PGMAXPROTOCOLVERSION", "PGSSLSNI", "PGSSLNEGOTIATION", "PGCHANNELBINDING", "PGREQUIREAUTH"} {
		if present(name) {
			return true
		}
	}
	return false
}
