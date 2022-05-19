package virt

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
)

func TestGetOsinfos(t *testing.T) {
	ls, err := GetOsinfos()
	assert.Nil(t, err)
	assert.NotEmpty(t, ls)

	for _, v := range ls {
		spew.Dump(v)
		//fmt.Printf("os: %+v\n", v)
	}
}
