package virt

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUndefineVm(t *testing.T) {
	name := "xxx"
	err := VmUndefine(name)
	assert.Nil(t, err)
}

func TestReloadVm(t *testing.T) {
	name := "xxx"
	err := VmReload(name)
	assert.Nil(t, err)
}

func TestVmDefine(t *testing.T) {
	caps, _ := GetDomainCaps("/usr/bin/qemu-system-x86_64", "x86_64", "q35")

	opt := &VmOption{
		Name:      "xxx",
		Desc:      "xxx_desc",
		OsVariant: "centos7.0",
		OsFamily:  "linux",
		Arch:      "x86_64",
		Machine:   "q35",
		Autostart: false,
		Memory:    4096,
		Vcpu:      4,
		CpuMode:   "host-model",
		Boot: &BootOption{
			Loader: "uefi",
		},
		ClockOffset: "localtime",
		Graphics: &GraphicsOption{
			Type:     "vnc",
			Port:     -1,
			Listen:   "0.0.0.0",
			Password: "xxx",
		},
		Video: &VideoOption{
			Model: "qxl",
		},
		Disks: []*DiskOption{
			{
				Device:    "cdrom",
				Path:      "/opt/mark/TrueNAS-SCALE-22.02.0.iso",
				Bus:       "sata",
				BootOrder: 1,
			},
			{
				Device: "cdrom",
				Path:   "/opt/mark/TrueNAS-SCALE-22.02.0.iso",
				Bus:    "sata",
			},
			{
				Device: "disk",
				Path:   "v1.qcow2",
				Bus:    "sata",
				Size:   4,
			},
			{
				Device: "disk",
				Path:   "v2.qcow2",
				Bus:    "sata",
				Size:   4,
			},
		},
		Nics: []*NicOption{
			{
				SourceType:  "bridge",
				SourceValue: "virbr0",
				Mac:         "52:54:00:f6:b8:9e",
				Model:       "virtio",
			},
		},
		IsSupportVirtio: true,
		domainCaps:      caps,
	}

	s, err := VmDefine(opt)
	assert.Nil(t, err)
	assert.NotEmpty(t, s)

	//spew.Dump(s)
	fmt.Println(s)

	vm, err := libvirtConn.DomainDefineXML(s)
	assert.Nil(t, err)
	assert.NotNil(t, vm)
	//fmt.Println(vm.Save(opt.Name + ".xml"))
	vm.Destroy()
}