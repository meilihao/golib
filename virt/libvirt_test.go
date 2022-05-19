package virt

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCap(t *testing.T) {
	caps, err := GetHostCaps()
	assert.Nil(t, err)
	assert.NotEmpty(t, caps)

	//spew.Dump(caps)
}

func TestGetDomainCaps(t *testing.T) {
	caps, err := GetDomainCaps("/usr/bin/qemu-system-x86_64", "x86_64", "q35")
	assert.Nil(t, err)
	assert.NotEmpty(t, caps)

	fmt.Println(caps)
}
