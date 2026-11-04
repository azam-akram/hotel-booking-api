package model

import (
	"fmt"
	"time"
)

const dateLayout = "2006-01-02"

// Date is a calendar date written as YYYY-MM-DD, with no time and no zone.
// It is used for check-in and check-out so the API never deals with time zones.
type Date time.Time

func (d Date) Time() time.Time {
	return time.Time(d)
}

func (d Date) IsZero() bool {
	return time.Time(d).IsZero()
}

func (d Date) String() string {
	return time.Time(d).Format(dateLayout)
}

// Nights returns the number of nights from d (check-in) to checkOut.
func (d Date) Nights(checkOut Date) int {
	return int(checkOut.Time().Sub(d.Time()).Hours() / 24)
}

// MarshalText makes JSON write the date as "2030-01-10".
func (d Date) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

// UnmarshalText reads a YYYY-MM-DD string. JSON bodies and query strings
// both use it.
func (d *Date) UnmarshalText(b []byte) error {
	t, err := time.Parse(dateLayout, string(b))
	if err != nil {
		return fmt.Errorf("invalid date %q, expected YYYY-MM-DD", b)
	}
	*d = Date(t)
	return nil
}

// UnmarshalParam lets Gin read a date from a query string. Gin does not use
// UnmarshalText for query values, so without this the dates fail to bind.
func (d *Date) UnmarshalParam(param string) error { return d.UnmarshalText([]byte(param)) }
