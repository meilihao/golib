package pager

import (
	"fmt"
	"testing"
)

func TestNewPager(t *testing.T) {
	cases := []struct {
		page     int
		size     int
		wantSize int
	}{
		{0, 0, 10},
		{1, 11, 11},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("pager(%d,%d)", c.page, c.size), func(t *testing.T) {
			p := NewPager(c.page, c.size)

			if p.Size != c.wantSize {
				t.Errorf("NewPager(%d,%d) failed", c.page, c.size)
			}
		})
	}
}
