package connections

import "regexp"

// WorkerLogins is trusted runtime configuration; HTTP requests cannot supply it.
type WorkerLogins struct {
	Sync, Publisher string
}

var workerLoginPattern = regexp.MustCompile(`^[a-zA-Z0-9_.@-]{1,63}$`)

// NewForBookCreators enables bounded creator onboarding using already provisioned SQL logins.
func NewForBookCreators(db Beginner, provider Provider, secrets SecretStore, logins WorkerLogins) (*Service, error) {
	if !workerLoginPattern.MatchString(logins.Sync) || !workerLoginPattern.MatchString(logins.Publisher) || logins.Sync == logins.Publisher {
		return nil, failed("configuration_error")
	}
	s := New(db, provider, secrets)
	s.creatorLogins = &logins
	return s, nil
}
