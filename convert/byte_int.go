package convert

import (
	"bytes"
	"encoding/binary"
	"strconv"
)

func IntToBytes(n int, order binary.ByteOrder) []byte {
	l := 8
	if (32 << (^uint(0) >> 63)) == 32 {
		l = 4
	}

	bytebuf := bytes.NewBuffer(make([]byte, l))
	binary.Write(bytebuf, order, n)
	return bytebuf.Bytes()
}

func StrToInt64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func StrToUint64(s string) uint64 {
	n, _ := strconv.ParseUint(s, 10, 64)
	return n
}
