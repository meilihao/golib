package virt

import (
	"errors"
	"regexp"
)

type OsFamily string

const (
	OsFamilyWinnt OsFamily = "winnt"
	OsFamilyLinux OsFamily = "linux"
)

const (
	BusUsb    = "usb"
	BusIde    = "ide"
	BusSata   = "sata"
	BusScsi   = "scsi"
	BusVirtio = "virtio"
)

const (
	ArchX86     = "x86"
	ArchAarch64 = "aarch64"
)

const (
	FirmwareBios = "bios"
	FirmwareUefi = "uefi"
)

const (
	DiskDeviceCdrom = "cdrom"
	DiskDeviceDisk  = "disk"
)

const (
	ClockOffsetUtc   = "utc"
	ClockOffsetLocal = "localtime"
)

var (
	reVmName  = regexp.MustCompile(`[a-zA-Z0-9_\-:]{1,64}`)
	reMacAddr = regexp.MustCompile(`^([0-9a-fA-F]{1,2}:){5}[0-9a-fA-F]{1,2}$`)
)

var (
	ErrVmNameInvalid = errors.New(`allow "a-z,A-Z,0-9,_,-,:" and not all numbers and len is 1~64`)
)
