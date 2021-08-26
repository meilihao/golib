package rand

import (
	crand "crypto/rand"
	"encoding/hex"
	mrand "math/rand"
	"time"
)

// n == 0, return ""
func Rand(n int) string {
	buf := make([]byte, n)

	var tmp int
	for n > 0 {
		tmp, _ = crand.Read(buf)
		if tmp == n {
			break
		}
	}

	return hex.EncodeToString(buf)
}

// n == 0, return ""
func RandNumber(l int) string {
	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))

	b := make([]byte, 0, l)
	for i := 0; i < l; i++ {
		b = append(b, 48+byte(r.Intn(10)))
	}

	return string(b)
}

const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func RandAll(n int) string {
	buf := make([]byte, n)
	max := len(alphanum)

	for i := 0; i < n; i++ {
		buf[i] = alphanum[RandInt(max)]
	}

	return string(buf)
}

// Intn returns, as an int, a non-negative pseudo-random number in [0,n).
// It panics if n <= 0.
func RandInt(n int) int {
	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))

	return r.Intn(n)
}
