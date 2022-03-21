package udev

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromName(t *testing.T) {
	d, err := FromName("block", "sda1")
	assert.Nil(t, err)
	assert.NotNil(t, d)

	fmt.Printf("%+v\n", d.Device)
	fmt.Println(d.Device.GetSubsystem())
	fmt.Println(d.Device.GetIdFilename())
}
