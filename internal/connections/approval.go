package connections

import "time"

// NewForBookCreatorsWithApproval fences provider work with an explicit immutable local deadline.
// Nil provider/store are permitted for cold expired startup; SQL-only operations retain current authority.
func NewForBookCreatorsWithApproval(db Beginner, provider Provider, secrets SecretStore, logins WorkerLogins, deadline time.Time, now func() time.Time) (*Service, error) {
	if db == nil || deadline.IsZero() || deadline.Year() < 1 || deadline.Year() > 9999 || now == nil {
		return nil, failed("configuration_error")
	}
	s, err := NewForBookCreators(db, provider, secrets, logins)
	if err != nil {
		return nil, err
	}
	s.providerApproved = func() bool { return deadline.After(now()) }
	return s, nil
}
