package bgen

import (
	"fmt"
	"time"
)

// Time exists to facilitate time parsing from the Metadata, because BGEN
// uses both unixtime and text strings to represent time. Derived from
// https://github.com/mattn/go-sqlite3/issues/190#issuecomment-343341834f
type Time time.Time

func (t *Time) Scan(v interface{}) error {
	switch which := v.(type) {
	case int64:
		vt := time.Unix(which, 0)
		*t = Time(vt)
		return nil
	case int:
		vt := time.Unix(int64(which), 0)
		*t = Time(vt)
		return nil
	case []byte:
		// Should be more strictly to check this type.
		vt, err := time.Parse("2006-01-02 15:04:05", string(which))
		if err != nil {
			return err
		}
		*t = Time(vt)
		return nil
	}

	return fmt.Errorf("No appropriate type could be found to decode %v", v)
}
