package time

import (
	"strings"
	"time"
)

type Mytime time.Time

const ctLayout = "2006-01-02 15:04:05"

// UnmarshalJSON Parses the json string in the custom format
func (ct *Mytime) UnmarshalJSON(b []byte) (err error) {
	if string(b) == "null" {
		return nil
	}

	s := strings.Trim(string(b), `"`)
	nt, err := time.Parse(ctLayout, s)
	*ct = Mytime(nt)
	return
}

// MarshalJSON writes a quoted string in the custom format
func (ct Mytime) MarshalJSON() ([]byte, error) {
	return []byte(ct.String()), nil
}

// String returns the time in the custom format
func (ct *Mytime) String() string {
	t := time.Time(*ct)
	return `"` + t.Format(ctLayout) + `"`
}
