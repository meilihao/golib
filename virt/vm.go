package virt

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/meilihao/golib/v2/cmd"
	"github.com/meilihao/golib/v2/file"
	"github.com/meilihao/golib/v2/misc"
)

/*
   vcpus = sa.Column(sa.Integer(), default=1)

   cores = sa.Column(sa.Integer(), default=1)
   threads = sa.Column(sa.Integer(), default=1)
   hide_from_msr = sa.Column(sa.Boolean(), default=False)
   ensure_display_device = sa.Column(sa.Boolean(), default=True)
   arch_type = sa.Column(sa.String(255), default=None, nullable=True)
   machine_type = sa.Column(sa.String(255), default=None, nullable=True)
*/
type DiskOption struct {
	Device    string
	Bus       string // 经验证: xp安装os成功后(已安装virio blk)的系统盘不能从ide转为virtio, 推测是os启动时没有加载virtio驱动, 但新增virtio disk是可以的.
	Path      string // virt-install 会检查是否已被使用
	Cache     string
	Size      uint32 //G, 当Path不存在时, size必须指定
	BootOrder uint16 // 启动磁盘必须设置BootOrder, 否则iso安装系统重启后, 选择本地磁盘启动会找不到启动盘
}

func (opt *DiskOption) Build(osFamily OsFamily, osVariant string, ideNo, scsiNo, virtioNo *DiskFromNumber) string {
	ops := make([]string, 0, 4)

	ops = append(ops, opt.Path)
	ops = append(ops, "device="+opt.Device)
	ops = append(ops, "bus="+opt.Bus)
	if opt.BootOrder > 0 {
		ops = append(ops, fmt.Sprintf("boot.order=%d", opt.BootOrder))
	}

	switch opt.Bus {
	case BusIde:
		ops = append(ops, "target.dev="+ideNo.Generate())
	case BusSata, BusScsi:
		ops = append(ops, "target.dev="+scsiNo.Generate())
	case BusVirtio:
		ops = append(ops, "target.dev="+virtioNo.Generate())
	}

	switch opt.Device {
	case DiskDeviceCdrom:
		ops = append(ops, "readonly=true")
	case DiskDeviceDisk:
		if opt.Size > 0 {
			ops = append(ops, fmt.Sprintf("size=%d", opt.Size))
		}
		if opt.Cache != "" {
			ops = append(ops, "cache="+opt.Cache)
		}
	}

	return strings.Join(ops, ",")
}

// 指定图形显示相关的配置
type GraphicsOption struct {
	Type     string
	Port     int32
	Listen   string // 0.0.0.0
	Password string
}

func (opt *GraphicsOption) Build() string {
	ops := make([]string, 0, 4)
	ops = append(ops, opt.Type)
	if opt.Password != "" {
		ops = append(ops, "password="+opt.Password)
	}
	ops = append(ops, "listen="+opt.Listen)
	if opt.Port == -1 {
		ops = append(ops, "port='-1'")
	} else {
		ops = append(ops, fmt.Sprintf("port=%d", opt.Port))
	}
	return strings.Join(ops, ",")
}

type NicOption struct {
	SourceType  string
	SourceValue string
	Mac         string // virt-install 会检查是否已被使用
	Model       string
	BootOrder   uint16
}

func (opt *NicOption) Build(isSupportVirtio bool) string {
	ops := make([]string, 0, 4)

	if opt.SourceType == "none" {
		return "none"
	}

	switch opt.SourceType {
	case "bridge":
		ops = append(ops, "bridge="+opt.SourceValue)
	case "network":
		ops = append(ops, "networok="+opt.SourceValue)
	}

	if opt.Mac == "" {
		opt.Mac = GenerateRandomMac()
	}
	ops = append(ops, "mac="+opt.Mac)

	if isSupportVirtio {
		opt.Model = BusVirtio
	}
	ops = append(ops, "model="+opt.Model)

	if opt.BootOrder > 0 {
		ops = append(ops, fmt.Sprintf("boot.order=%d", opt.BootOrder))
	}
	return strings.Join(ops, ",")
}

func (opt *NicOption) Validate() error {
	if !reMacAddr.MatchString(opt.Mac) {
		return fmt.Errorf("invalid mac: %s", opt.Mac)
	}

	return nil
}

// 显卡
type VideoOption struct {
	Model string // qxl,bochs
}

func (opt *VideoOption) Build() string {
	return "model=" + opt.Model
}

type SoundhwOption struct {
	Model string
}

// virt-install --boot help
type BootOption struct {
	Firmware string
	BootMenu bool

	Loader       string
	LoaderSecure string
	LoaderType   string
}

func (opt *BootOption) Build() string {
	opt.BootMenu = true

	ops := make([]string, 0, 4)
	if opt.Firmware == FirmwareUefi {
		ops = append(ops, FirmwareUefi)

		if opt.Loader != "" {
			ops = append(ops, "loader="+opt.Loader)
			ops = append(ops, "loader.readonly=yes")
		}
		if opt.LoaderType != "" {
			ops = append(ops, "loader.type="+opt.LoaderType)
		}
		if opt.LoaderSecure != "" {
			ops = append(ops, "loader.secure="+opt.LoaderSecure)
		}
	}
	if opt.BootMenu {
		ops = append(ops, "bootmenu.enable=true")
	}

	return strings.Join(ops, ",")
}

func (opt *BootOption) Validate(parent *VmOption) error {
	if opt == nil || !(opt.Firmware == FirmwareUefi || opt.Firmware == FirmwareBios) {
		return errors.New("invalid boot firmware")
	}

	if opt.Firmware == FirmwareBios {
		return nil
	}

	f := parent.domainCaps.UEFIFirmwares()
	if f == nil {
		return errors.New("failed to get uefi firmware support")
	}

	return f.Validate(opt.Loader, opt.LoaderType, opt.LoaderSecure)
}

/*
--vcpus 5
--vcpus 5,maxvcpus=10,cpuset=1-4,6,8
--vcpus sockets=2,cores=4,threads=2
*/
type VmOption struct {
	Name            string
	Desc            string
	OsVariant       string
	OsFamily        OsFamily // linux, winnt
	Arch            string
	Machine         string // aarch64=virt, x64=q35
	CpuMode         string
	CpuModel        string
	Autostart       bool
	Memory          int64 // MB
	Vcpu            uint32
	Boot            *BootOption // uefi,mbr
	ClockOffset     string      // utc/localtime
	Graphics        *GraphicsOption
	Video           *VideoOption
	Soundhw         *SoundhwOption
	Disks           []*DiskOption
	Nics            []*NicOption
	IsSupportVirtio bool
	domainCaps      *DomainCaps
}

func (opt *VmOption) Validate() error {
	if !IsKvmOkForGuest(opt.Arch) {
		return errors.New("unsupport kvm in guest arch")
	}

	if !reVmName.MatchString(opt.Name) || misc.IsAllNumbers(opt.Name) {
		return ErrVmNameInvalid
	}
	if len(opt.Desc) > 255 {
		return errors.New("invalid desc")
	}

	if !ValidateOsinfo(opt.OsFamily, opt.OsVariant) {
		return errors.New("invalid osvariant")
	}

	if !ValidateArchMachine(opt.Arch, opt.Machine) {
		return errors.New("invalid arch or machine")
	}

	if !(opt.ClockOffset == ClockOffsetLocal || opt.ClockOffset == ClockOffsetUtc) {
		return errors.New("invalid clock")
	}

	if opt.Vcpu > uint32(opt.domainCaps.VcpuMax()) {
		return fmt.Errorf("over cpu max: %d", opt.domainCaps.VcpuMax())
	}

	if len(opt.Nics) == 0 {
		return errors.New("no nic")
	}

	if len(opt.Disks) == 0 {
		return errors.New("no disk")
	}

	var err error
	if err = opt.Boot.Validate(opt); err != nil {
		return err
	}

	for _, v := range opt.Nics {
		if err = v.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (opt *VmOption) Convert() error {
	if opt.OsVariant == "winxp" || opt.OsVariant == "win2k" {
		opt.Machine = "pc" // q35 acpi version太高, xp bluescreen
	}

	if opt.Arch == ArchAarch64 {
		opt.Boot.Firmware = FirmwareUefi
	}

	for _, v := range opt.Disks {
		if opt.OsFamily == OsFamilyWinnt && (opt.OsVariant == "winxp" || opt.OsVariant == "win2k") {
			v.Bus = BusIde
		}
	}

	return nil
}

func BuildVirtIntall(opt *VmOption) string {
	ops := make([]string, 0, 15)
	ops = append(ops, "virt-install --dry-run --print-xml")
	ops = append(ops, "--name="+opt.Name)
	if opt.Desc != "" {
		ops = append(ops, "--description="+opt.Desc)
	}
	ops = append(ops, "--os-variant="+opt.OsVariant)
	ops = append(ops, fmt.Sprintf("--memory %d", opt.Memory))
	ops = append(ops, fmt.Sprintf("--vcpus %d", opt.Vcpu))
	if opt.CpuMode == "host-passthrough" {
		ops = append(ops, "--cpu=host-passthrough")
	} else if opt.CpuMode == "host-model" {
		ops = append(ops, "--cpu=host-model")
	} else {
		ops = append(ops, "--cpu=qemu64")
	}
	ops = append(ops, "--arch="+opt.Arch)

	ops = append(ops, "--machine="+opt.Machine)
	if opt.Soundhw != nil {
		ops = append(ops, "--soundhw "+opt.Soundhw.Model)
	}

	ops = append(ops, "--clock offset="+opt.ClockOffset)
	ops = append(ops, "--graphics "+opt.Graphics.Build())
	ops = append(ops, "--video "+opt.Video.Build())

	ops = append(ops, "--boot "+opt.Boot.Build())

	for _, v := range opt.Nics {
		ops = append(ops, "--network "+v.Build(opt.IsSupportVirtio))
	}

	var inputBus string
	diskBus := opt.domainCaps.DiskBus()
	if misc.IsInStrings(BusUsb, diskBus) {
		inputBus = BusUsb
	}
	if inputBus == "" && opt.IsSupportVirtio {
		inputBus = BusVirtio
	}
	if strings.Contains(opt.Arch, ArchX86) {
		// 加tablet防止出现鼠标漂移
		inputBus = BusUsb // 用virtio还是有较大漂移在xp上
		ops = append(ops, "--input type=tablet,bus="+inputBus)
		// input ps2是默认设备, 没法删除, 即使删除, libvirtd也会自动添加
		ops = append(ops, "--input type=mouse")
		ops = append(ops, "--input type=keyboard")
	} else {
		ops = append(ops, "--input type=mouse,bus="+inputBus)
		ops = append(ops, "--input type=keyboard,bus="+inputBus)
		//	ops = append(ops, "--input type=tablet,bus="+inputBus)
	}

	ideNo := NewDiskFromNumber(BusIde, 1)
	virtioNo := NewDiskFromNumber(BusVirtio, 1)
	scsiNo := NewDiskFromNumber(BusScsi, 1)
	for _, v := range opt.Disks {
		ops = append(ops, "--disk "+v.Build(opt.OsFamily, opt.OsVariant, ideNo, scsiNo, virtioNo))
	}

	ops = append(ops, "--check disk_size=off")

	return strings.Join(ops, " ")
}

// virt-install 自动添加pci control
func VmDefine(opt *VmOption) (string, error) {
	caps := GetDomainCapsFromCache(GetEmulatorByArch(opt.Arch), opt.Arch, opt.Machine)
	if caps == nil {
		return "", errors.New("missing domain caps")
	}
	opt.domainCaps = caps

	err := opt.Convert()
	if err != nil {
		return "", err
	}
	err = opt.Validate()
	if err != nil {
		return "", err
	}

	s := BuildVirtIntall(opt)

	copt := &cmd.Option{}
	out, err := cmd.CmdCombinedBashWithCtx(context.TODO(), copt, s)
	if err != nil {
		return "", err
	}

	tmp := string(out)
	if i := strings.Index(s, "<domain type"); i != -1 {
		tmp = tmp[i:]
	}

	return tmp, nil
}

const (
	ErrSflagNoDomain = "Domain not found"
)

func VmUndefine(name string) error {
	dom, err := libvirtConn.LookupDomainByName(name)
	if err != nil {
		if strings.Contains(err.Error(), ErrSflagNoDomain) {
			return nil
		}

		return err
	}
	defer dom.Free()

	return dom.Undefine()
}

func VmReload(name string) error {
	p := fmt.Sprintf("/etc/libvirt/qemu/%s.xml", name)
	if !file.IsFile(p) {
		return errors.New(ErrSflagNoDomain)
	}

	copt := &cmd.Option{}
	_, err := cmd.CmdCombinedBashWithCtx(context.TODO(), copt,
		fmt.Sprintf(" virsh define %s", p),
	)
	if err != nil {
		return err
	}

	return nil
}
