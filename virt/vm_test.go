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

func TestVmDefinePreXml(t *testing.T) {
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
			Firmware: "uefi",
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
				Type:       "bridge",
				Source:     "virbr0",
				SourceMode: "",
				Mac:        "52:54:00:f6:b8:9e",
				Model:      "virtio",
			},
		},
		IsSupportVirtio: true,
	}

	s, err := VmDefinePreXml(opt)
	assert.Nil(t, err)
	assert.NotEmpty(t, s)

	//spew.Dump(s)
	fmt.Println(s)

	vm, err := libvirtConn.DomainDefineXML(s)
	assert.Nil(t, err)
	assert.NotNil(t, vm)
	//fmt.Println(vm.Save(opt.Name + ".xml"))
	vm.Free()
}

func TestAddDisk(t *testing.T) {
	r := &AddDiskReq{
		Domain: "xxx",
		Disk: &DiskOption{
			Path:   "/mnt/1.qcow2",
			Device: DiskDeviceDisk,
			Bus:    BusSata,
			//TargetDev: "sdc",
		},
		IsHotunplug: false,
	}
	err := AddDisk(r)
	assert.Nil(t, err)
}

func TestRemoveDisk(t *testing.T) {
	r := &RemoveDiskReq{
		Domain:    "xxx",
		TargetDev: "vdc",
	}
	err := RemoveDisk(r)
	assert.Nil(t, err)
}

func TestAddNic(t *testing.T) {
	r := &AddNicReq{
		Domain: "xxx",
		Nic: &NicOption{
			Type:       "bridge",
			Source:     "virbr0",
			SourceMode: "",
			Mac:        "52:54:00:f6:b8:9f",
			Model:      "virtio",
		},
		IsHotunplug: false,
	}
	err := AddNic(r)
	assert.Nil(t, err)
}

func TestRemoveNic(t *testing.T) {
	r := &RemoveNicReq{
		Domain: "xxx",
		Mac:    "52:54:00:f6:b8:9f",
	}
	err := RemoveNic(r)
	assert.Nil(t, err)
}

func TestVmChangeCommon(t *testing.T) {
	r := &VmChangeCommonReq{
		Domain: "xxx",
		Memory: 3072,
		Vcpu:   3,
	}
	err := VmChangeCommon(r)
	assert.Nil(t, err)
}
