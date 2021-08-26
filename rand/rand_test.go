package rand

import (
	"fmt"
	"regexp"
	"testing"
)

func TestRand(t *testing.T) {
	cases := []struct {
		l    int
		want int
	}{
		{0, 0},
		{4, 8},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%d", c.l), func(t *testing.T) {
			s := Rand(c.l)
			fmt.Println(s)

			if len(s) != c.want {
				t.Errorf("Rand(%d) got %d, want %d", c.l, len(s), c.want)
			}
		})
	}
}

func TestRandNumber(t *testing.T) {
	cases := []struct {
		l    int
		want int
	}{
		{0, 0},
		{4, 4},
	}

	for _, c := range cases {
		r := regexp.MustCompile(fmt.Sprintf(`\d{%d}`, c.l))

		t.Run(fmt.Sprintf("%d", c.l), func(t *testing.T) {
			s := RandNumber(c.l)
			fmt.Println(s, r.MatchString(s))

			if len(s) != c.want || !r.MatchString(s) {
				t.Errorf("RandNumber(%d) got %d, want %d", c.l, len(s), c.want)
			}
		})
	}
}

func TestRandAll(t *testing.T) {
	cases := []struct {
		l    int
		want int
	}{
		{0, 0},
		{4, 4},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%d", c.l), func(t *testing.T) {
			s := RandAll(c.l)
			fmt.Println(s)

			if len(s) != c.want {
				t.Errorf("RandAll(%d) got %d, want %d", c.l, len(s), c.want)
			}
		})
	}
}

func TestRandInt(t *testing.T) {
	cases := []struct {
		l    int
		want int
	}{
		{1, 1},
		{4, 4},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%d", c.l), func(t *testing.T) {
			s := RandInt(c.l)
			fmt.Println(s)

			if s >= c.want {
				t.Errorf("RandInt(%d) got %d, want lt %d", c.l, s, c.want)
			}
		})
	}
}
