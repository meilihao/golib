package time

import (
	"strings"
	"time"
)

type Int64 int64

func (t Int64) MarshalJSON() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t *Int64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	if string(data) == `""` {
		*t = 0
		return nil
	}

	s := strings.Trim(string(data), `"`)
	n, err := time.ParseInLocation(ctLayout, s, time.Local)
	*t = Int64(n.Unix())
	return err
}

func (t *Int64) String() string {
	if *t == 0 {
		return `""`
	}

	n := time.Unix(int64(*t), 0)
	return `"` + n.Local().Format(ctLayout) + `"`
}
