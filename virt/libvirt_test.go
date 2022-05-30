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

func TestGetDomainCapsInfo(t *testing.T) {
	info, err := GetDomainCapsInfo("/usr/bin/qemu-system-x86_64", "x86_64", "q35", "centos-stream9")
	assert.Nil(t, err)
	assert.NotEmpty(t, info)

	fmt.Println(info)
}

func TestLibvirtApi(t *testing.T) {
	data, err := libvirtConn.GetMaxVcpus("KVM")
	if err != nil {
		return
	}
	fmt.Println(data)
}
