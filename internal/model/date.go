package model

import (
	"fmt"
	"time"
)

const dateLayout = "2006-01-02"

// Date is a calendar date (no time, no zone) in the form YYYY-MM-DD.
// It is used for check-in and check-out so the API never deals with time zones.
type Date time.Time

// ParseDate parses a YYYY-MM-DD string.
func ParseDate(s string) (Date, error) {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q, expected YYYY-MM-DD", s)
	}
	return Date(t), nil
}

// Time returns the date as a time.Time at midnight UTC.
func (d Date) Time() time.Time { return time.Time(d) }

// IsZero reports whether the date is unset.
func (d Date) IsZero() bool { return time.Time(d).IsZero() }

// String returns the date as YYYY-MM-DD.
func (d Date) String() string { return time.Time(d).Format(dateLayout) }

// Nights returns the number of nights between d (check-in) and checkOut.
func (d Date) Nights(checkOut Date) int {
	return int(checkOut.Time().Sub(d.Time()).Hours() / 24)
}

// MarshalText implements encoding.TextMarshaler (used by JSON encoding).
func (d Date) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

// UnmarshalText implements encoding.TextUnmarshaler. It is used by JSON
// decoding.
func (d *Date) UnmarshalText(b []byte) error {
	return d.UnmarshalParam(string(b))
}

// UnmarshalParam implements Gin's BindUnmarshaler for query parameters. Without
// it Gin treats Date (a struct kind) as JSON and rejects a bare 2030-01-10.
func (d *Date) UnmarshalParam(s string) error {
	parsed, err := ParseDate(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}
