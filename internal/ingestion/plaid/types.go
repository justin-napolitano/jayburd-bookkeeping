// Package plaid decodes bounded synthetic Transactions sync pages without network or ledger authority.
package plaid

const (
	Profile    = "jayburd-plaid-transactions/1"
	APIVersion = "2020-09-14"
	MaxBytes   = 2 << 20
	MaxChanges = 500
)

// Currency is trusted installed ISO metadata, never inferred from a provider balance.
type Currency struct {
	Code      string
	VersionID string
	MinorUnit int
	ValidFrom string
	ValidTo   string // Empty means no exclusive end date.
}

// Expected comes from the approved account-filtered stream, not provider or queue input.
type Expected struct {
	AccountID  string
	Currencies []Currency
}

// Transaction preserves exact source values. EligibleForReview is source readiness only, not authorization.
type Transaction struct {
	AccountID            string  `json:"account_id"`
	TransactionID        string  `json:"transaction_id"`
	Pending              bool    `json:"pending"`
	PendingTransactionID *string `json:"pending_transaction_id"`
	Date                 string  `json:"date"`
	AuthorizedDate       *string `json:"authorized_date"`
	Datetime             *string `json:"datetime"`
	AuthorizedDatetime   *string `json:"authorized_datetime"`
	ISOCurrencyCode      string  `json:"iso_currency_code"`
	CurrencyVersionID    string  `json:"currency_version_id"`
	AmountLexeme         string  `json:"amount_lexeme"`
	Amount               string  `json:"amount"`
	Direction            string  `json:"direction"`
	EligibleForReview    bool    `json:"eligible_for_review"`
	Name                 string  `json:"name"`
	MerchantName         *string `json:"merchant_name"`
	OriginalDescription  *string `json:"original_description"`
	Raw                  []byte  `json:"-"`
}

// Removed is a tombstone identity; no amount, matching decision or journal is invented.
type Removed struct {
	AccountID     string `json:"account_id"`
	TransactionID string `json:"transaction_id"`
	Raw           []byte `json:"-"`
}

// Page preserves provider order and bytes; HasMore=false does not authorize publication or imply history readiness.
type Page struct {
	NextCursor   string        `json:"next_cursor"`
	HasMore      bool          `json:"has_more"`
	UpdateStatus string        `json:"transactions_update_status"`
	RequestID    string        `json:"request_id"`
	SHA256       string        `json:"sha256"`
	Added        []Transaction `json:"added"`
	Modified     []Transaction `json:"modified"`
	Removed      []Removed     `json:"removed"`
	Raw          []byte        `json:"-"`
}

// Error contains one fixed classification; raw bytes, IDs and parser details are never exposed.
type Error struct{ Code string }

// Error exposes only the fixed classification.
func (e *Error) Error() string { return e.Code }

// fail constructs a redacted failure with no parser or source data.
func fail(code string) error { return &Error{Code: code} }
