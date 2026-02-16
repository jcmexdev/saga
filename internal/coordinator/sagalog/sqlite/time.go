package sqlite

import (
	"fmt"
	"time"
)

// parseRFC3339 parses the timestamp strings stored in SQLite.
// SQLite has no native datetime type; we store RFC3339 TEXT.
func parseRFC3339(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("sqlite: parse time %q: %w", s, err)
	}
	return t, nil
}
