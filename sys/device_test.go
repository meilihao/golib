package udev

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
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

	p, err := d.Device.GetParent()
	assert.Nil(t, err)
	assert.NotNil(t, p)

	fmt.Printf("%+v\n", *p)
}

func TestFromNameForGpu(t *testing.T) {
	re := regexp.MustCompile(`(?m)VGA compatible controller|Display controller|3D controller`)

	cmd := exec.Command("lspci", "-D")
	output, err := cmd.CombinedOutput()
	assert.Nil(t, err)

	var addr string
	lines := strings.Split(string(output), "\n")
	for i := range lines {
		if re.MatchString(lines[i]) {
			idx := strings.Index(lines[i], " ")
			addr = lines[i][:idx]
		}
	}
	if addr == "" {
		return
	}
	fmt.Println("gpu addr:", addr)

	d, err := FromName("pci", addr)
	assert.Nil(t, err)
	assert.NotNil(t, d)

	fmt.Printf("%+v\n", d.Device)
	fmt.Println(d.Device.GetSubsystem())
	fmt.Println(d.Device.GetIdFilename())
}

func TestListDevices(t *testing.T) {
	ds, err := ListDevices("block", WithFilterDevtype("disk"))
	assert.Nil(t, err)
	assert.NotNil(t, ds)

	fmt.Printf("%+v\n", ds[0])
}
