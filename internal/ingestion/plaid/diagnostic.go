package plaid

import "errors"

// ErrorCode exposes only reviewed classifications, even for malformed injected errors.
func ErrorCode(err error) string {
	var value *Error
	if errors.As(err, &value) && value != nil {
		switch value.Code {
		case "invalid_json", "duplicate_key", "limit_exceeded", "invalid_number", "invalid_expectation", "invalid_field", "unknown_field", "unknown_enum", "invalid_cursor_encoding", "invalid_cursor_progress", "account_mismatch", "unsupported_account", "invalid_identity", "duplicate_identity", "invalid_lineage", "invalid_date", "unsupported_currency", "amount_out_of_range":
			return value.Code
		}
	}
	return "invalid_field"
}
