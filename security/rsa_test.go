package security

import (
	"encoding/base64"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
)

func TestRSA(t *testing.T) {
	num := 1
	bits := 2048

	err := InitRSA(bits, num)
	assert.Nil(t, err)

	p := GetRSAByRandom()
	assert.NotEmpty(t, p)

	spew.Dump(p.Index, base64.StdEncoding.EncodeToString(p.Public.N.Bytes()))

	msg := []byte("hello rsa!")

	eData, err := p.Encrypt(msg)
	assert.Nil(t, err)
	assert.NotEmpty(t, eData)
	spew.Dump(eData)

	p2 := GetRSAByIndex(p.Index)
	assert.NotEmpty(t, p2)
	assert.Equal(t, p2, p)

	dData, err := p2.Decrypt(eData)
	assert.Nil(t, err)
	assert.Equal(t, msg, dData)
	spew.Dump(dData)
}
