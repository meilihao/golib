package virt

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/meilihao/golib/v2/file"
	"github.com/meilihao/golib/v2/log"
	"github.com/meilihao/golib/v2/misc"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"libvirt.org/go/libvirt"
	"libvirt.org/go/libvirtxml"
)

var (
	LibvirtdUri = "qemu:///system"
	libvirtConn *libvirt.Connect

	GlobalHostCapsLock sync.Mutex
	GlobalHostCapsErr  error
	GlobalHostCaps     *HostCaps

	GlobalDomainCapsLock sync.RWMutex
	GlobalDomainCaps     = map[string]*DomainCaps{}
)

func init() {
	var err error
	if !file.IsDir(LibvirtAutostartPath) {
		if err = os.MkdirAll(LibvirtAutostartPath, 0755); err != nil {
			log.Glog.Panic("libvirt create autostart", zap.Error(err))
		}
	}

	libvirtConn, err = libvirt.NewConnect(LibvirtdUri)
	if err != nil {
		log.Glog.Panic("libvirt conn", zap.Error(err))
	}

	LoadHostCaps()
	if GlobalHostCapsErr != nil {
		log.Glog.Panic(GlobalHostCapsErr.Error())
	}
}

func LoadHostCaps() {
	GlobalHostCapsLock.Lock()
	defer GlobalHostCapsLock.Unlock()

	GlobalHostCaps, GlobalHostCapsErr = GetHostCaps()
	if GlobalHostCapsErr != nil {
		GlobalHostCaps = nil
	}
}

func ValidateArchMachine(arch, machine string) bool {
	ls := GlobalHostCaps.ListGuests()
	for _, v := range ls {
		if v.Arch == arch {
			for j := range v.Machines {
				if v.Machines[j] == machine {
					return true
				}
			}
		}
	}

	return false
}

func GetEmulatorByArch(arch string) string {
	ls := GlobalHostCaps.ListGuests()
	for _, v := range ls {
		if v.Arch == arch {
			return v.Emulator
		}
	}

	return ""
}

func IsKvmOkForGuest(arch string) bool {
	ls := GlobalHostCaps.ListGuests()
	for _, v := range ls {
		if v.Arch == arch {
			return misc.IsInStrings("kvm", v.Domains)
		}
	}

	return false
}

func GetDomainCapsFromCache(emulatorbin, arch, machine string) (caps *DomainCaps) {
	fn := func(emulatorbin, arch, machine string) string {
		return arch + "|" + machine + "|" + emulatorbin
	}
	k := fn(emulatorbin, arch, machine)

	GlobalDomainCapsLock.RLock()

	caps = GlobalDomainCaps[k]
	if caps != nil {
		GlobalDomainCapsLock.RUnlock()
		return
	}
	GlobalDomainCapsLock.RUnlock()

	GlobalDomainCapsLock.Lock()
	defer GlobalDomainCapsLock.Unlock()

	var err error
	caps, err = GetDomainCaps(emulatorbin, arch, machine)
	if err != nil {
		log.Glog.Error("", zap.Error(err))
		return
	}

	GlobalDomainCaps[k] = caps

	return
}

// func GetConnection() (*libvirt.Connect, error) {
// 	conn, err := libvirt.NewConnect(LibvirtdUri)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return conn, nil
// }

type HostCaps struct {
	*libvirtxml.Caps
}

type Guest struct {
	Arch     string   `json:"arch"`
	Emulator string   `json:"emulator"`
	Domains  []string `json:"domains"`
	Machines []string `json:"machines"`
}

func (c *HostCaps) ListGuests() []*Guest {
	ls := make([]*Guest, 0, len(c.Guests))
	for _, v := range c.Guests {
		tmp := &Guest{
			Arch:     v.Arch.Name,
			Emulator: v.Arch.Emulator,
			Machines: make([]string, 0, len(v.Arch.Machines)),
			Domains:  make([]string, 0, len(v.Arch.Domains)),
		}

		for _, m := range v.Arch.Machines {
			tmp.Machines = append(tmp.Machines, m.Name)
		}
		for _, d := range v.Arch.Domains {
			tmp.Domains = append(tmp.Domains, d.Type)
		}

		ls = append(ls, tmp)
	}

	return ls
}

func GetHostCaps() (caps *HostCaps, err error) {
	// virsh capabilities
	data, err := libvirtConn.GetCapabilities()
	if err != nil {
		return
	}

	//data:
	/*
		<capabilities>

			<host>
				<uuid>03000200-0400-0500-0006-000700080009</uuid>
				<cpu>
				<arch>x86_64</arch>
				<model>Nehalem-IBRS</model>
				<vendor>Intel</vendor>
				<microcode version='2316'/>
				<counter name='tsc' frequency='2000000000' scaling='no'/>
				<topology sockets='1' dies='1' cores='4' threads='1'/>
				<feature name='vme'/>
				<feature name='ds'/>
				<feature name='acpi'/>
				<feature name='ss'/>
				<feature name='ht'/>
				<feature name='tm'/>
				<feature name='pbe'/>
				<feature name='pclmuldq'/>
				<feature name='dtes64'/>
				<feature name='monitor'/>
				<feature name='ds_cpl'/>
				<feature name='vmx'/>
				<feature name='est'/>
				<feature name='tm2'/>
				<feature name='xtpr'/>
				<feature name='pdcm'/>
				<feature name='movbe'/>
				<feature name='tsc-deadline'/>
				<feature name='rdrand'/>
				<feature name='arat'/>
				<feature name='tsc_adjust'/>
				<feature name='smep'/>
				<feature name='erms'/>
				<feature name='md-clear'/>
				<feature name='stibp'/>
				<feature name='rdtscp'/>
				<feature name='3dnowprefetch'/>
				<feature name='invtsc'/>
				<pages unit='KiB' size='4'/>
				<pages unit='KiB' size='2048'/>
				</cpu>
				<power_management>
				<suspend_mem/>
				</power_management>
				<iommu support='no'/>
				<migration_features>
				<live/>
				<uri_transports>
					<uri_transport>tcp</uri_transport>
					<uri_transport>rdma</uri_transport>
				</uri_transports>
				</migration_features>
				<topology>
				<cells num='1'>
					<cell id='0'>
					<memory unit='KiB'>7960336</memory>
					<pages unit='KiB' size='4'>1990084</pages>
					<pages unit='KiB' size='2048'>0</pages>
					<distances>
						<sibling id='0' value='10'/>
					</distances>
					<cpus num='4'>
						<cpu id='0' socket_id='0' die_id='0' core_id='0' siblings='0'/>
						<cpu id='1' socket_id='0' die_id='0' core_id='1' siblings='1'/>
						<cpu id='2' socket_id='0' die_id='0' core_id='2' siblings='2'/>
						<cpu id='3' socket_id='0' die_id='0' core_id='3' siblings='3'/>
					</cpus>
					</cell>
				</cells>
				</topology>
				<secmodel>
				<model>apparmor</model>
				<doi>0</doi>
				</secmodel>
				<secmodel>
				<model>dac</model>
				<doi>0</doi>
				<baselabel type='kvm'>+64055:+109</baselabel>
				<baselabel type='qemu'>+64055:+109</baselabel>
				</secmodel>
			</host>

			<guest>
				<os_type>hvm</os_type>
				<arch name='i686'>
				<wordsize>32</wordsize>
				<emulator>/usr/bin/qemu-system-i386</emulator>
				<machine maxCpus='255'>pc-i440fx-jammy</machine>
				<machine canonical='pc-i440fx-jammy' maxCpus='255'>ubuntu</machine>
				<machine maxCpus='255'>pc-i440fx-impish-hpb</machine>
				<machine maxCpus='288'>pc-q35-5.2</machine>
				<machine maxCpus='255'>pc-i440fx-2.12</machine>
				<machine maxCpus='255'>pc-i440fx-2.0</machine>
				<machine maxCpus='255'>pc-i440fx-xenial</machine>
				<machine maxCpus='255'>pc-i440fx-6.2</machine>
				<machine canonical='pc-i440fx-6.2' maxCpus='255'>pc</machine>
				<machine maxCpus='288'>pc-q35-4.2</machine>
				<machine maxCpus='255'>pc-i440fx-2.5</machine>
				<machine maxCpus='255'>pc-i440fx-4.2</machine>
				<machine maxCpus='255'>pc-i440fx-focal</machine>
				<machine maxCpus='255'>pc-i440fx-hirsute</machine>
				<machine maxCpus='255'>pc-q35-xenial</machine>
				<machine maxCpus='255'>pc-i440fx-jammy-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-5.2</machine>
				<machine maxCpus='255'>pc-i440fx-1.5</machine>
				<machine maxCpus='255'>pc-q35-2.7</machine>
				<machine maxCpus='288'>pc-q35-eoan-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-zesty</machine>
				<machine maxCpus='255'>pc-i440fx-disco-hpb</machine>
				<machine maxCpus='288'>pc-q35-groovy</machine>
				<machine maxCpus='255'>pc-i440fx-groovy</machine>
				<machine maxCpus='288'>pc-q35-artful</machine>
				<machine maxCpus='255'>pc-i440fx-2.2</machine>
				<machine maxCpus='255'>pc-i440fx-trusty</machine>
				<machine maxCpus='255'>pc-i440fx-eoan-hpb</machine>
				<machine maxCpus='288'>pc-q35-focal-hpb</machine>
				<machine maxCpus='288'>pc-q35-bionic-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-artful</machine>
				<machine maxCpus='255'>pc-i440fx-2.7</machine>
				<machine maxCpus='288'>pc-q35-6.1</machine>
				<machine maxCpus='255'>pc-i440fx-yakkety</machine>
				<machine maxCpus='255'>pc-q35-2.4</machine>
				<machine maxCpus='288'>pc-q35-cosmic-hpb</machine>
				<machine maxCpus='288'>pc-q35-2.10</machine>
				<machine maxCpus='1'>x-remote</machine>
				<machine maxCpus='288'>pc-q35-5.1</machine>
				<machine maxCpus='255'>pc-i440fx-1.7</machine>
				<machine maxCpus='288'>pc-q35-2.9</machine>
				<machine maxCpus='255'>pc-i440fx-2.11</machine>
				<machine maxCpus='288'>pc-q35-3.1</machine>
				<machine maxCpus='255'>pc-i440fx-6.1</machine>
				<machine maxCpus='288'>pc-q35-4.1</machine>
				<machine maxCpus='288'>pc-q35-jammy</machine>
				<machine canonical='pc-q35-jammy' maxCpus='288'>ubuntu-q35</machine>
				<machine maxCpus='255'>pc-i440fx-2.4</machine>
				<machine maxCpus='255'>pc-i440fx-4.1</machine>
				<machine maxCpus='288'>pc-q35-eoan</machine>
				<machine maxCpus='288'>pc-q35-jammy-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-5.1</machine>
				<machine maxCpus='255'>pc-i440fx-2.9</machine>
				<machine maxCpus='255'>pc-i440fx-bionic-hpb</machine>
				<machine maxCpus='1'>isapc</machine>
				<machine maxCpus='255'>pc-i440fx-1.4</machine>
				<machine maxCpus='288'>pc-q35-cosmic</machine>
				<machine maxCpus='255'>pc-q35-2.6</machine>
				<machine maxCpus='255'>pc-i440fx-3.1</machine>
				<machine maxCpus='288'>pc-q35-bionic</machine>
				<machine maxCpus='288'>pc-q35-disco-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-cosmic</machine>
				<machine maxCpus='288'>pc-q35-2.12</machine>
				<machine maxCpus='255'>pc-i440fx-bionic</machine>
				<machine maxCpus='288'>pc-q35-groovy-hpb</machine>
				<machine maxCpus='288'>pc-q35-disco</machine>
				<machine maxCpus='255'>pc-i440fx-cosmic-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-2.1</machine>
				<machine maxCpus='255'>pc-i440fx-wily</machine>
				<machine maxCpus='288'>pc-q35-impish</machine>
				<machine maxCpus='288'>pc-q35-6.0</machine>
				<machine maxCpus='255'>pc-i440fx-impish</machine>
				<machine maxCpus='255'>pc-i440fx-2.6</machine>
				<machine maxCpus='288'>pc-q35-impish-hpb</machine>
				<machine maxCpus='288'>pc-q35-hirsute</machine>
				<machine maxCpus='288'>pc-q35-4.0.1</machine>
				<machine maxCpus='288'>pc-q35-hirsute-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-1.6</machine>
				<machine maxCpus='288'>pc-q35-5.0</machine>
				<machine maxCpus='288'>pc-q35-2.8</machine>
				<machine maxCpus='255'>pc-i440fx-2.10</machine>
				<machine maxCpus='288'>pc-q35-3.0</machine>
				<machine maxCpus='255'>pc-i440fx-6.0</machine>
				<machine maxCpus='288'>pc-q35-zesty</machine>
				<machine maxCpus='288'>pc-q35-4.0</machine>
				<machine maxCpus='288'>pc-q35-focal</machine>
				<machine maxCpus='288'>microvm</machine>
				<machine maxCpus='255'>pc-i440fx-2.3</machine>
				<machine maxCpus='255'>pc-i440fx-focal-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-disco</machine>
				<machine maxCpus='255'>pc-i440fx-4.0</machine>
				<machine maxCpus='255'>pc-i440fx-groovy-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-hirsute-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-5.0</machine>
				<machine maxCpus='288'>pc-q35-6.2</machine>
				<machine canonical='pc-q35-6.2' maxCpus='288'>q35</machine>
				<machine maxCpus='255'>pc-i440fx-2.8</machine>
				<machine maxCpus='255'>pc-i440fx-eoan</machine>
				<machine maxCpus='255'>pc-q35-2.5</machine>
				<machine maxCpus='255'>pc-i440fx-3.0</machine>
				<machine maxCpus='255'>pc-q35-yakkety</machine>
				<machine maxCpus='288'>pc-q35-2.11</machine>
				<domain type='qemu'/>
				<domain type='kvm'/>
				</arch>
				<features>
				<pae/>
				<nonpae/>
				<acpi default='on' toggle='yes'/>
				<apic default='on' toggle='no'/>
				<cpuselection/>
				<deviceboot/>
				<disksnapshot default='on' toggle='no'/>
				</features>
			</guest>

			<guest>
				<os_type>hvm</os_type>
				<arch name='x86_64'>
				<wordsize>64</wordsize>
				<emulator>/usr/bin/qemu-system-x86_64</emulator>
				<machine maxCpus='255'>pc-i440fx-jammy</machine>
				<machine canonical='pc-i440fx-jammy' maxCpus='255'>ubuntu</machine>
				<machine maxCpus='255'>pc-i440fx-impish-hpb</machine>
				<machine maxCpus='288'>pc-q35-5.2</machine>
				<machine maxCpus='255'>pc-i440fx-2.12</machine>
				<machine maxCpus='255'>pc-i440fx-2.0</machine>
				<machine maxCpus='255'>pc-i440fx-xenial</machine>
				<machine maxCpus='255'>pc-i440fx-6.2</machine>
				<machine canonical='pc-i440fx-6.2' maxCpus='255'>pc</machine>
				<machine maxCpus='288'>pc-q35-4.2</machine>
				<machine maxCpus='255'>pc-i440fx-2.5</machine>
				<machine maxCpus='255'>pc-i440fx-4.2</machine>
				<machine maxCpus='255'>pc-i440fx-hirsute</machine>
				<machine maxCpus='255'>pc-i440fx-focal</machine>
				<machine maxCpus='255'>pc-q35-xenial</machine>
				<machine maxCpus='255'>pc-i440fx-jammy-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-5.2</machine>
				<machine maxCpus='255'>pc-i440fx-1.5</machine>
				<machine maxCpus='255'>pc-q35-2.7</machine>
				<machine maxCpus='288'>pc-q35-eoan-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-zesty</machine>
				<machine maxCpus='255'>pc-i440fx-disco-hpb</machine>
				<machine maxCpus='288'>pc-q35-groovy</machine>
				<machine maxCpus='255'>pc-i440fx-groovy</machine>
				<machine maxCpus='288'>pc-q35-artful</machine>
				<machine maxCpus='255'>pc-i440fx-trusty</machine>
				<machine maxCpus='255'>pc-i440fx-2.2</machine>
				<machine maxCpus='288'>pc-q35-focal-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-eoan-hpb</machine>
				<machine maxCpus='288'>pc-q35-bionic-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-artful</machine>
				<machine maxCpus='255'>pc-i440fx-2.7</machine>
				<machine maxCpus='288'>pc-q35-6.1</machine>
				<machine maxCpus='255'>pc-i440fx-yakkety</machine>
				<machine maxCpus='255'>pc-q35-2.4</machine>
				<machine maxCpus='288'>pc-q35-cosmic-hpb</machine>
				<machine maxCpus='288'>pc-q35-2.10</machine>
				<machine maxCpus='1'>x-remote</machine>
				<machine maxCpus='288'>pc-q35-5.1</machine>
				<machine maxCpus='255'>pc-i440fx-1.7</machine>
				<machine maxCpus='288'>pc-q35-2.9</machine>
				<machine maxCpus='255'>pc-i440fx-2.11</machine>
				<machine maxCpus='288'>pc-q35-3.1</machine>
				<machine maxCpus='255'>pc-i440fx-6.1</machine>
				<machine maxCpus='288'>pc-q35-4.1</machine>
				<machine maxCpus='288'>pc-q35-jammy</machine>
				<machine canonical='pc-q35-jammy' maxCpus='288'>ubuntu-q35</machine>
				<machine maxCpus='255'>pc-i440fx-2.4</machine>
				<machine maxCpus='255'>pc-i440fx-4.1</machine>
				<machine maxCpus='288'>pc-q35-eoan</machine>
				<machine maxCpus='288'>pc-q35-jammy-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-5.1</machine>
				<machine maxCpus='255'>pc-i440fx-2.9</machine>
				<machine maxCpus='255'>pc-i440fx-bionic-hpb</machine>
				<machine maxCpus='1'>isapc</machine>
				<machine maxCpus='255'>pc-i440fx-1.4</machine>
				<machine maxCpus='288'>pc-q35-cosmic</machine>
				<machine maxCpus='255'>pc-q35-2.6</machine>
				<machine maxCpus='255'>pc-i440fx-3.1</machine>
				<machine maxCpus='288'>pc-q35-bionic</machine>
				<machine maxCpus='288'>pc-q35-disco-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-cosmic</machine>
				<machine maxCpus='288'>pc-q35-2.12</machine>
				<machine maxCpus='255'>pc-i440fx-bionic</machine>
				<machine maxCpus='288'>pc-q35-groovy-hpb</machine>
				<machine maxCpus='288'>pc-q35-disco</machine>
				<machine maxCpus='255'>pc-i440fx-cosmic-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-2.1</machine>
				<machine maxCpus='255'>pc-i440fx-wily</machine>
				<machine maxCpus='288'>pc-q35-impish</machine>
				<machine maxCpus='255'>pc-i440fx-2.6</machine>
				<machine maxCpus='288'>pc-q35-6.0</machine>
				<machine maxCpus='255'>pc-i440fx-impish</machine>
				<machine maxCpus='288'>pc-q35-impish-hpb</machine>
				<machine maxCpus='288'>pc-q35-hirsute</machine>
				<machine maxCpus='288'>pc-q35-4.0.1</machine>
				<machine maxCpus='288'>pc-q35-hirsute-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-1.6</machine>
				<machine maxCpus='288'>pc-q35-5.0</machine>
				<machine maxCpus='288'>pc-q35-2.8</machine>
				<machine maxCpus='255'>pc-i440fx-2.10</machine>
				<machine maxCpus='288'>pc-q35-3.0</machine>
				<machine maxCpus='288'>pc-q35-zesty</machine>
				<machine maxCpus='288'>pc-q35-4.0</machine>
				<machine maxCpus='288'>pc-q35-focal</machine>
				<machine maxCpus='288'>microvm</machine>
				<machine maxCpus='255'>pc-i440fx-6.0</machine>
				<machine maxCpus='255'>pc-i440fx-2.3</machine>
				<machine maxCpus='255'>pc-i440fx-disco</machine>
				<machine maxCpus='255'>pc-i440fx-focal-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-4.0</machine>
				<machine maxCpus='255'>pc-i440fx-groovy-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-hirsute-hpb</machine>
				<machine maxCpus='255'>pc-i440fx-5.0</machine>
				<machine maxCpus='255'>pc-i440fx-2.8</machine>
				<machine maxCpus='288'>pc-q35-6.2</machine>
				<machine canonical='pc-q35-6.2' maxCpus='288'>q35</machine>
				<machine maxCpus='255'>pc-i440fx-eoan</machine>
				<machine maxCpus='255'>pc-q35-2.5</machine>
				<machine maxCpus='255'>pc-i440fx-3.0</machine>
				<machine maxCpus='255'>pc-q35-yakkety</machine>
				<machine maxCpus='288'>pc-q35-2.11</machine>
				<domain type='qemu'/>
				<domain type='kvm'/>
				</arch>
				<features>
				<acpi default='on' toggle='yes'/>
				<apic default='on' toggle='no'/>
				<cpuselection/>
				<deviceboot/>
				<disksnapshot default='on' toggle='no'/>
				</features>
			</guest>

			</capabilities>
	*/
	tmp := new(libvirtxml.Caps)
	if err = tmp.Unmarshal(data); err != nil {
		err = errors.Wrap(err, "failed to decode libvirtd capabilities")
		return
	}

	return &HostCaps{Caps: tmp}, nil
}

type HostCapsInfo struct {
	Guests []*Guest `json:"guests"`
}

func GetHostCapsInfo() (*HostCapsInfo, error) {
	return &HostCapsInfo{
		Guests: GlobalHostCaps.ListGuests(),
	}, nil
}

type DomainCaps struct {
	*libvirtxml.DomainCaps
}

func (c *DomainCaps) DiskBus() []string {
	if c.Devices.Disk.Supported != "yes" {
		return []string{}
	}

	for _, v := range c.Devices.Disk.Enums {
		if v.Name == "bus" {
			return misc.ExcludeStrings(v.Values, []string{"fdc"})
		}
	}

	return nil
}

func (c *DomainCaps) Videos() []string {
	if c.Devices.Video.Supported != "yes" {
		return []string{}
	}

	for _, v := range c.Devices.Video.Enums {
		if v.Name == "modelType" {
			tmp := make([]string, 0, 3)
			for _, vv := range v.Values {
				if misc.IsInStrings(vv, []string{"qxl", "virtio", "vga"}) {
					tmp = append(tmp, vv)
				}
			}

			return tmp
		}
	}

	return nil
}

func (c *DomainCaps) Graphics() []string {
	if c.Devices.Graphics.Supported != "yes" {
		return []string{}
	}

	for _, v := range c.Devices.Graphics.Enums {
		if v.Name == "type" {
			tmp := make([]string, 0, 2)
			for _, vv := range v.Values {
				if misc.IsInStrings(vv, []string{"vnc", "spice"}) {
					tmp = append(tmp, vv)
				}
			}

			return tmp
		}
	}

	return nil
}

type UEFIFirmware struct {
	Loaders []string `json:"loaders"`
	Types   []string `json:"types"`
	Secures []string `json:"secures"`
}

func (f *UEFIFirmware) Validate(loader, typ, secure string) error {
	if loader != "" && !misc.IsInStrings(loader, f.Loaders) {
		return errors.New("invalid uefi loader")
	}
	if typ != "" && !misc.IsInStrings(typ, f.Types) {
		return errors.New("invalid uefi loader type")
	}
	if secure != "" && !misc.IsInStrings(secure, f.Secures) {
		return errors.New("invalid uefi loader secure")
	}

	return nil
}

func (c *DomainCaps) UEFIFirmwares() *UEFIFirmware {
	if c.OS.Supported != "yes" {
		return nil
	}

	for _, v := range c.OS.Enums {
		if v.Name == "firmware" && misc.IsInStrings("efi", v.Values) {
			if c.OS.Loader.Supported == "yes" {
				tmp := &UEFIFirmware{
					Loaders: c.OS.Loader.Values,
				}

				for _, e := range c.OS.Loader.Enums {
					if e.Name == "type" {
						if misc.IsInStrings("pflash", e.Values) {
							tmp.Types = []string{"pflash"}
						} else {
							tmp.Types = e.Values
						}
					}
					if e.Name == "secure" {
						tmp.Secures = e.Values
					}
				}

				return tmp
			}
		}
	}

	return nil
}

func (c *DomainCaps) VcpuMax() uint {
	return c.VCPU.Max
}

// only for kvm
func GetDomainCaps(emulatorbin, arch, machine string) (caps *DomainCaps, err error) {
	// virsh domcapabilities
	data, err := libvirtConn.GetDomainCapabilities(emulatorbin, arch, machine, "kvm", 0)
	if err != nil {
		return
	}

	// data(/usr/bin/qemu-system-x86_64, x86_64, q35,kvm,0):
	/*
		<domainCapabilities>
			<path>/usr/bin/qemu-system-x86_64</path>
			<domain>kvm</domain>
			<machine>pc-q35-6.2</machine>
			<arch>x86_64</arch>
			<vcpu max='288'/>
			<iothreads supported='yes'/>
			<os supported='yes'>
				<enum name='firmware'>
				<value>efi</value>
				</enum>
				<loader supported='yes'>
				<value>/usr/share/OVMF/OVMF_CODE_4M.ms.fd</value>
				<value>/usr/share/OVMF/OVMF_CODE_4M.secboot.fd</value>
				<value>/usr/share/OVMF/OVMF_CODE_4M.fd</value>
				<enum name='type'>
					<value>rom</value>
					<value>pflash</value>
				</enum>
				<enum name='readonly'>
					<value>yes</value>
					<value>no</value>
				</enum>
				<enum name='secure'>
					<value>yes</value>
					<value>no</value>
				</enum>
				</loader>
			</os>
			<cpu>
				<mode name='host-passthrough' supported='yes'>
				<enum name='hostPassthroughMigratable'>
					<value>on</value>
					<value>off</value>
				</enum>
				</mode>
				<mode name='maximum' supported='yes'>
				<enum name='maximumMigratable'>
					<value>on</value>
					<value>off</value>
				</enum>
				</mode>
				<mode name='host-model' supported='yes'>
				<model fallback='forbid'>IvyBridge-IBRS</model>
				<vendor>Intel</vendor>
				<feature policy='require' name='ss'/>
				<feature policy='require' name='vmx'/>
				<feature policy='require' name='pdcm'/>
				<feature policy='require' name='movbe'/>
				<feature policy='require' name='hypervisor'/>
				<feature policy='require' name='arat'/>
				<feature policy='require' name='tsc_adjust'/>
				<feature policy='require' name='umip'/>
				<feature policy='require' name='md-clear'/>
				<feature policy='require' name='stibp'/>
				<feature policy='require' name='arch-capabilities'/>
				<feature policy='require' name='3dnowprefetch'/>
				<feature policy='require' name='invtsc'/>
				<feature policy='require' name='ibpb'/>
				<feature policy='require' name='ibrs'/>
				<feature policy='require' name='amd-stibp'/>
				<feature policy='require' name='amd-no-ssb'/>
				<feature policy='require' name='skip-l1dfl-vmentry'/>
				<feature policy='require' name='ssb-no'/>
				<feature policy='require' name='pschange-mc-no'/>
				<feature policy='disable' name='aes'/>
				<feature policy='disable' name='xsave'/>
				<feature policy='disable' name='avx'/>
				<feature policy='disable' name='f16c'/>
				<feature policy='disable' name='fsgsbase'/>
				<feature policy='disable' name='xsaveopt'/>
				</mode>
				<mode name='custom' supported='yes'>
				<model usable='no'>qemu64</model>
				<model usable='yes'>qemu32</model>
				<model usable='no'>phenom</model>
				<model usable='yes'>pentium3</model>
				<model usable='yes'>pentium2</model>
				<model usable='yes'>pentium</model>
				<model usable='yes'>n270</model>
				<model usable='yes'>kvm64</model>
				<model usable='yes'>kvm32</model>
				<model usable='yes'>coreduo</model>
				<model usable='yes'>core2duo</model>
				<model usable='no'>athlon</model>
				<model usable='no'>Westmere-IBRS</model>
				<model usable='no'>Westmere</model>
				<model usable='no'>Snowridge</model>
				<model usable='no'>Skylake-Server-noTSX-IBRS</model>
				<model usable='no'>Skylake-Server-IBRS</model>
				<model usable='no'>Skylake-Server</model>
				<model usable='no'>Skylake-Client-noTSX-IBRS</model>
				<model usable='no'>Skylake-Client-IBRS</model>
				<model usable='no'>Skylake-Client</model>
				<model usable='no'>SandyBridge-IBRS</model>
				<model usable='no'>SandyBridge</model>
				<model usable='yes'>Penryn</model>
				<model usable='no'>Opteron_G5</model>
				<model usable='no'>Opteron_G4</model>
				<model usable='no'>Opteron_G3</model>
				<model usable='no'>Opteron_G2</model>
				<model usable='yes'>Opteron_G1</model>
				<model usable='yes'>Nehalem-IBRS</model>
				<model usable='yes'>Nehalem</model>
				<model usable='no'>IvyBridge-IBRS</model>
				<model usable='no'>IvyBridge</model>
				<model usable='no'>Icelake-Server-noTSX</model>
				<model usable='no'>Icelake-Server</model>
				<model usable='no' deprecated='yes'>Icelake-Client-noTSX</model>
				<model usable='no' deprecated='yes'>Icelake-Client</model>
				<model usable='no'>Haswell-noTSX-IBRS</model>
				<model usable='no'>Haswell-noTSX</model>
				<model usable='no'>Haswell-IBRS</model>
				<model usable='no'>Haswell</model>
				<model usable='no'>EPYC-Rome</model>
				<model usable='no'>EPYC-Milan</model>
				<model usable='no'>EPYC-IBPB</model>
				<model usable='no'>EPYC</model>
				<model usable='no'>Dhyana</model>
				<model usable='no'>Cooperlake</model>
				<model usable='yes'>Conroe</model>
				<model usable='no'>Cascadelake-Server-noTSX</model>
				<model usable='no'>Cascadelake-Server</model>
				<model usable='no'>Broadwell-noTSX-IBRS</model>
				<model usable='no'>Broadwell-noTSX</model>
				<model usable='no'>Broadwell-IBRS</model>
				<model usable='no'>Broadwell</model>
				<model usable='yes'>486</model>
				</mode>
			</cpu>
			<memoryBacking supported='yes'>
				<enum name='sourceType'>
				<value>file</value>
				<value>anonymous</value>
				<value>memfd</value>
				</enum>
			</memoryBacking>
			<devices>
				<disk supported='yes'>
				<enum name='diskDevice'>
					<value>disk</value>
					<value>cdrom</value>
					<value>floppy</value>
					<value>lun</value>
				</enum>
				<enum name='bus'>
					<value>fdc</value>
					<value>scsi</value>
					<value>virtio</value>
					<value>usb</value>
					<value>sata</value>
				</enum>
				<enum name='model'>
					<value>virtio</value>
					<value>virtio-transitional</value>
					<value>virtio-non-transitional</value>
				</enum>
				</disk>
				<graphics supported='yes'>
				<enum name='type'>
					<value>sdl</value>
					<value>vnc</value>
					<value>spice</value>
					<value>egl-headless</value>
				</enum>
				</graphics>
				<video supported='yes'>
				<enum name='modelType'>
					<value>vga</value>
					<value>cirrus</value>
					<value>vmvga</value>
					<value>qxl</value>
					<value>virtio</value>
					<value>none</value>
					<value>bochs</value>
					<value>ramfb</value>
				</enum>
				</video>
				<hostdev supported='yes'>
				<enum name='mode'>
					<value>subsystem</value>
				</enum>
				<enum name='startupPolicy'>
					<value>default</value>
					<value>mandatory</value>
					<value>requisite</value>
					<value>optional</value>
				</enum>
				<enum name='subsysType'>
					<value>usb</value>
					<value>pci</value>
					<value>scsi</value>
				</enum>
				<enum name='capsType'/>
				<enum name='pciBackend'/>
				</hostdev>
				<rng supported='yes'>
				<enum name='model'>
					<value>virtio</value>
					<value>virtio-transitional</value>
					<value>virtio-non-transitional</value>
				</enum>
				<enum name='backendModel'>
					<value>random</value>
					<value>egd</value>
					<value>builtin</value>
				</enum>
				</rng>
				<filesystem supported='yes'>
				<enum name='driverType'>
					<value>path</value>
					<value>handle</value>
					<value>virtiofs</value>
				</enum>
				</filesystem>
				<tpm supported='yes'>
				<enum name='model'>
					<value>tpm-tis</value>
					<value>tpm-crb</value>
				</enum>
				<enum name='backendModel'>
					<value>passthrough</value>
				</enum>
				</tpm>
			</devices>
			<features>
				<gic supported='no'/>
				<vmcoreinfo supported='yes'/>
				<genid supported='yes'/>
				<backingStoreInput supported='yes'/>
				<backup supported='yes'/>
				<sev supported='no'/>
			</features>
			</domainCapabilities>
	*/
	tmp := new(libvirtxml.DomainCaps)
	if err = tmp.Unmarshal(data); err != nil {
		err = errors.Wrap(err, "failed to decode domain capabilities")
		return
	}

	return &DomainCaps{DomainCaps: tmp}, nil
}

type DomainCapsInfo struct {
	IsSupportVirtio bool          `json:"isSupportVirtio"`
	VcpuMax         uint          `json:"vcpuMax"`
	UEFI            *UEFIFirmware `json:"uefi"`
	Videos          []string      `json:"videos"`
	Graphics        []string      `json:"graphics"`
	DiskBus         []string      `json:"diskBus"`
	Nics            []string      `json:"nics"`
	Sounds          []string      `json:"sounds"`
}

func GetDomainCapsInfo(emulatorbin, arch, machine, osVariant string) (*DomainCapsInfo, error) {
	caps := GetDomainCapsFromCache(emulatorbin, arch, machine)
	if caps == nil {
		return nil, errors.New("no domain caps")
	}

	info := &DomainCapsInfo{
		VcpuMax:  caps.VcpuMax(),
		UEFI:     caps.UEFIFirmwares(),
		Videos:   caps.Videos(),
		Graphics: caps.Graphics(),
		DiskBus:  caps.DiskBus(),
		Nics:     GetNicModels(emulatorbin, arch, machine, osVariant),
		Sounds:   []string{"ich6", "ich9", "ac97"}, // from virtmanager: sound_recommended_models
	}
	info.IsSupportVirtio = IsSupportsVirtio(arch, machine) && misc.IsInStrings("virtio", info.DiskBus)

	return info, nil
}

// virtmanager: interface_recommended_models
func GetNicModels(emulatorbin, arch, machine, osVariant string) []string {
	ls := []string{"defualt"}
	if IsSupportsVirtionetByOsVariant(osVariant) || IsSupportsVirtio(arch, machine) {
		ls = append(ls, "virtio")
	}

	if !(arch == "i686" || arch == "x86_64") {
		return ls
	}

	if strings.Contains(machine, "q35") {
		ls = append(ls, "e1000e")
	} else {
		ls = append(ls, "e1000")
	}

	return ls
}

func IsSupportsVirtionetByOsVariant(osVariant string) bool {
	var id string
	for i := range GlobalOsinfos {
		if GlobalOsinfos[i].ShortId == osVariant {
			id = GlobalOsinfos[i].Id
			break
		}
	}

	if id == "" {
		return false
	}

	u, err := url.Parse(id)
	if err != nil {
		return false
	}

	tmp := strings.Replace(strings.TrimPrefix(u.Path, "/"), "/", "-", 1)
	data := file.FileValue(filepath.Join("/usr/share/osinfo/os", u.Host, tmp+".xml"))

	// virtio-net/virtio1.0-net
	return strings.Contains(data, "http://pcisig.com/pci/1af4/1000") || strings.Contains(data, "http://pcisig.com/pci/1af4/1041")
}
