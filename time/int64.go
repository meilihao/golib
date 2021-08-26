package time

import (
	"strings"
	"time"
)

const (
	f1 = "2006-01-02 15:04:05"
)

type Int64 int64

func (t Int64) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

func (t *Int64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	s := strings.Trim(string(data), `"`)
	n, err := time.ParseInLocation(f1, s, time.Local)
	*t = Int64(n.Unix())
	return err
}

func (t *Int64) String() string {
	n := time.Unix(int64(*t), 0)
	return n.Local().Format(f1)
}
