package virt

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/meilihao/golib/v2/cmd"
	"github.com/meilihao/golib/v2/file"
	"github.com/meilihao/golib/v2/log"
	"github.com/meilihao/golib/v2/misc"
	"go.uber.org/zap"
	"libvirt.org/go/libvirt"
	"libvirt.org/go/libvirtxml"
)

/*
兼容性virt-install 1.x:
1. boot.order -> boot_order
1. target.dev -> target
1. bootmenu -> 没有
*/

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
	Device    string `json:"device" binding:"required"`
	Bus       string `json:"bus" binding:"required"`  // found: os installed for xp(virtio driver installed too), can‘t change boot disk form ide to virtio, maybe need some steps like update initramfs with linux. but new disk can use virtio
	Path      string `json:"path" binding:"required"` // virt-install will check used
	TargetDev string `json:"targetDev"`
	Cache     string `josn:"cache"`
	Size      uint32 `json:"size"`      // GB. when path isn't exist, size is must
	BootOrder uint16 `json:"bootOrder"` // boot disk need BootOrder, otherwise vm isn't found boot disk after os installed
}

func (opt *DiskOption) Build(ideNo, scsiNo, virtioNo *DiskFromNumber) string {
	ops := make([]string, 0, 4)

	ops = append(ops, opt.Path)
	ops = append(ops, "device="+opt.Device)
	ops = append(ops, "bus="+opt.Bus)
	if opt.BootOrder > 0 {
		ops = append(ops, fmt.Sprintf("boot.order=%d", opt.BootOrder))
	}

	switch opt.Bus {
	case BusIde:
		if opt.TargetDev != "" {
			ops = append(ops, "target.dev="+opt.TargetDev)
		} else {
			ops = append(ops, "target.dev="+ideNo.Generate())
		}
	case BusSata, BusScsi:
		if opt.TargetDev != "" {
			ops = append(ops, "target.dev="+opt.TargetDev)
		} else {
			ops = append(ops, "target.dev="+scsiNo.Generate())
		}
	case BusVirtio:
		if opt.TargetDev != "" {
			ops = append(ops, "target.dev="+opt.TargetDev)
		} else {
			ops = append(ops, "target.dev="+virtioNo.Generate())
		}
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

func (opt *DiskOption) Validate() error {
	if opt.Device == DiskDeviceDisk && opt.TargetDev != "" && !IsValidDiskName(opt.Bus, opt.TargetDev) {
		return errors.New("name isn't match bus")
	}

	return nil
}

// 指定图形显示相关的配置
type GraphicsOption struct {
	Type     string `json:"type" binding:"required"`
	Port     int32  `json:"port" binding:"required"`   // -1
	Listen   string `json:"listen" binding:"required"` // 0.0.0.0
	Password string `json:"password"`
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
	SourceType  string `json:"sourceTyp" binding:"required"`
	SourceValue string `json:"sourceValue" binding:"required"`
	Mac         string `json:"mac" binding:"required"` // virt-install will check is used
	Model       string `json:"model" binding:"required"`
	BootOrder   uint16 `json:"bootOrder"`
}

func (opt *NicOption) Build() string {
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

type VideoOption struct {
	Model string `json:"model" binding:"required"` // qxl,bochs
}

func (opt *VideoOption) Build() string {
	return "model=" + opt.Model
}

type SoundOption struct {
	Model string `json:"model" binding:"required"`
}

// virt-install --boot help
type BootOption struct {
	Firmware string `json:"firmware" binding:"required"`
	BootMenu bool   `json:"bootMenu"`

	Loader       string `json:"loader"`
	LoaderSecure string `json:"loaderSecure"`
	LoaderType   string `json:"loaderType"`
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
	Name            string          `json:"name" binding:"required"`
	Desc            string          `json:"desc"`
	OsVariant       string          `json:"osVariant" binding:"required"`
	OsFamily        OsFamily        `json:"osFamily" binding:"required"` // linux, winnt
	Arch            string          `json:"arch" binding:"required"`
	Machine         string          `json:"machine" binding:"required"` // aarch64=virt, x64=q35
	CpuMode         string          `json:"cpuMode" binding:"required"`
	CpuModel        string          `json:"cpuModel"`
	Autostart       bool            `json:"autostart"`
	Memory          uint64          `json:"memory" binding:"required"` // MB
	Vcpu            uint            `json:"vcpu" binding:"required"`
	Boot            *BootOption     `json:"boot" binding:"required"`        // uefi,mbr
	ClockOffset     string          `json:"clockOffset" binding:"required"` // utc/localtime
	Graphics        *GraphicsOption `json:"graphics"  binding:"required"`
	Video           *VideoOption    `json:"video"  binding:"required"`
	Sound           *SoundOption    `json:"sound"`
	Disks           []*DiskOption   `json:"disks"  binding:"required"`
	Nics            []*NicOption    `json:"nics"  binding:"required"`
	IsDryRun        bool            `json:"isDryRun"`
	IsSupportVirtio bool            `json:"-"`
	domainCaps      *DomainCaps     `json:"-"`
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

	if opt.Vcpu > opt.domainCaps.VcpuMax() {
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

	var foundBootDevice bool
	for _, v := range opt.Disks {
		if err = v.Validate(); err != nil {
			return err
		}

		if !foundBootDevice && v.BootOrder > 0 {
			foundBootDevice = true
		}
	}

	for _, v := range opt.Nics {
		if err = v.Validate(); err != nil {
			return err
		}

		if !foundBootDevice && v.BootOrder > 0 {
			foundBootDevice = true
		}
	}
	if !foundBootDevice {
		return errors.New("missing boot device")
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
		if v.Device == DiskDeviceCdrom {
			v.Bus = BusSata
		}

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
	ops = append(ops, fmt.Sprintf(fmt.Sprintf("--memory memory=%d,maxmemory=%d", opt.Memory, opt.Memory)))
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
	if opt.Sound != nil {
		ops = append(ops, "--soundhw "+opt.Sound.Model)
	}

	ops = append(ops, "--clock offset="+opt.ClockOffset)
	ops = append(ops, "--graphics "+opt.Graphics.Build())
	ops = append(ops, "--video "+opt.Video.Build())

	ops = append(ops, "--boot "+opt.Boot.Build())

	for _, v := range opt.Nics {
		ops = append(ops, "--network "+v.Build())
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
		// input ps2 is default device, libvirtd will add it auto when deleted
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
		ops = append(ops, "--disk "+v.Build(ideNo, scsiNo, virtioNo))
	}

	ops = append(ops, "--check disk_size=off")

	return strings.Join(ops, " ")
}

// virt-install will auto insert pci control
func VmDefinePreXml(opt *VmOption) (string, error) {
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
	if i := strings.Index(tmp, "<domain type"); i != -1 {
		tmp = tmp[i:]
	}

	return tmp, nil
}

func VmCreate(opt *VmOption) error {
	s, err := VmDefinePreXml(opt)
	if err != nil {
		return err
	}

	if opt.IsDryRun {
		log.Glog.Info("vm create dryrun", zap.String("xml", s))
		return nil
	}

	vm, err := libvirtConn.DomainDefineXML(s)
	if err != nil {
		return err
	}
	vm.Free()

	if opt.Autostart {
		p := fmt.Sprintf(LibvritVmXmlVarPath, opt.Name)
		os.Symlink(p, filepath.Join(LibvirtAutostartPath, filepath.Base(p)))
	}

	return nil
}

const (
	ErrSDomainNotFound = "Domain not found"
)

var (
	ErrDomainNotFound = errors.New(ErrSDomainNotFound)
)

func VmUndefine(name string) error {
	dom, err := libvirtConn.LookupDomainByName(name)
	if err != nil {
		if strings.Contains(err.Error(), ErrSDomainNotFound) {
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
		return ErrDomainNotFound
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

func FindVm(name string) (*libvirt.Domain, error) {
	dom, err := libvirtConn.LookupDomainByName(name)
	if err != nil {
		return nil, ErrDomainNotFound
	}

	return dom, nil
}

type AddDiskReq struct {
	Domain      string      `json:"domain" binding:"required"`
	Disk        *DiskOption `json:"disk" binding:"required"`
	IsHotunplug bool        `json:"isHotunplug" binding:"required"`
}

func AddDisk(r *AddDiskReq) error {
	if err := r.Disk.Validate(); err != nil {
		return err
	}

	dom, err := FindVm(r.Domain)
	if err != nil {
		return err
	}
	defer dom.Free()

	state, _, err := dom.GetState()
	if err != nil {
		return err
	}

	if state == libvirt.DOMAIN_RUNNING || state == libvirt.DOMAIN_PAUSED {
		r.IsHotunplug = true
	}

	tmp, err := dom.GetXMLDesc(0)
	if err != nil {
		return err
	}

	doc := &libvirtxml.Domain{}
	if err = doc.Unmarshal(tmp); err != nil {
		return err
	}

	if r.Disk.TargetDev != "" {
		var isFound bool
		disks := doc.Devices.Disks
		for _, v := range disks {
			if v.Target.Dev == r.Disk.TargetDev {
				isFound = true
				break
			}
		}
		if isFound {
			return errors.New("disk is exsist")
		}
	}

	if r.Disk.TargetDev == "" {
		var curDiskList []string

		disks := doc.Devices.Disks
		for _, v := range disks {
			if v.Target.Bus == r.Disk.Bus {
				curDiskList = append(curDiskList, v.Target.Dev)
			}
		}

		var idNo *DiskFromNumber
		if len(curDiskList) == 0 {
			idNo = NewDiskFromNumber(r.Disk.Bus, 1)
		} else {
			idNo = NewDiskFromNumberByName(curDiskList[len(curDiskList)-1])
		}

		r.Disk.TargetDev = idNo.Generate()
	}

	parts := []string{
		"virt-xml",
		r.Domain,
	}
	if r.IsHotunplug {
		parts = append(parts, "--update")
	}
	parts = append(parts, "--add-device", "--disk", r.Disk.Build(nil, nil, nil))

	opt := &cmd.Option{}
	_, err = cmd.CmdCombinedBashWithCtx(context.TODO(), opt,
		strings.Join(parts, " "),
	)
	if err != nil {
		return err
	}

	return nil
}

type RemoveDiskReq struct {
	Domain    string `json:"domain" binding:"required"`
	TargetDev string `json:"targetDev" binding:"required"`
}

func RemoveDisk(r *RemoveDiskReq) error {
	dom, err := FindVm(r.Domain)
	if err != nil {
		return err
	}
	defer dom.Free()

	state, _, err := dom.GetState()
	if err != nil {
		return err
	}

	if state == libvirt.DOMAIN_RUNNING || state == libvirt.DOMAIN_PAUSED {
		return errors.New("vm is running or paused")
	}

	tmp, err := dom.GetXMLDesc(0)
	if err != nil {
		return err
	}

	doc := &libvirtxml.Domain{}
	if err = doc.Unmarshal(tmp); err != nil {
		return err
	}

	var isFound bool
	disks := doc.Devices.Disks
	for _, v := range disks {
		if v.Target.Dev == r.TargetDev {
			isFound = true
			break
		}
	}
	if !isFound {
		return errors.New("disk is't exsist")
	}

	parts := []string{
		"virt-xml",
		r.Domain,
	}
	parts = append(parts, "--remove-device", "--disk", "target="+r.TargetDev)

	opt := &cmd.Option{}
	_, err = cmd.CmdCombinedBashWithCtx(context.TODO(), opt,
		strings.Join(parts, " "),
	)
	if err != nil {
		return err
	}

	return nil
}

type AddNicReq struct {
	Domain      string     `json:"domain" binding:"required"`
	Nic         *NicOption `json:"nic" binding:"required"`
	IsHotunplug bool       `json:"isHotunplug" binding:"required"`
}

func AddNic(r *AddNicReq) error {
	if err := r.Nic.Validate(); err != nil {
		return err
	}

	dom, err := FindVm(r.Domain)
	if err != nil {
		return err
	}
	defer dom.Free()

	state, _, err := dom.GetState()
	if err != nil {
		return err
	}

	if state == libvirt.DOMAIN_RUNNING || state == libvirt.DOMAIN_PAUSED {
		r.IsHotunplug = true
	}

	tmp, err := dom.GetXMLDesc(0)
	if err != nil {
		return err
	}

	doc := &libvirtxml.Domain{}
	if err = doc.Unmarshal(tmp); err != nil {
		return err
	}

	if r.Nic.Mac != "" {
		var isFound bool
		nics := doc.Devices.Interfaces
		for _, v := range nics {
			if v.MAC.Address == r.Nic.Mac {
				isFound = true
				break
			}
		}
		if isFound {
			return errors.New("nic mac is exsist")
		}
	}

	if r.Nic.Mac == "" {
		r.Nic.Mac = GenerateRandomMac()
	}

	parts := []string{
		"virt-xml",
		r.Domain,
	}
	if r.IsHotunplug {
		parts = append(parts, "--update")
	}
	parts = append(parts, "--add-device", "--network", r.Nic.Build())

	opt := &cmd.Option{}
	_, err = cmd.CmdCombinedBashWithCtx(context.TODO(), opt,
		strings.Join(parts, " "),
	)
	if err != nil {
		return err
	}

	return nil
}

type RemoveNicReq struct {
	Domain string `json:"domain" binding:"required"`
	Mac    string `json:"mac" binding:"required"`
}

func RemoveNic(r *RemoveNicReq) error {
	dom, err := FindVm(r.Domain)
	if err != nil {
		return err
	}
	defer dom.Free()

	state, _, err := dom.GetState()
	if err != nil {
		return err
	}

	if state == libvirt.DOMAIN_RUNNING || state == libvirt.DOMAIN_PAUSED {
		return errors.New("vm is running or paused")
	}

	tmp, err := dom.GetXMLDesc(0)
	if err != nil {
		return err
	}

	doc := &libvirtxml.Domain{}
	if err = doc.Unmarshal(tmp); err != nil {
		return err
	}

	var isFound bool
	nics := doc.Devices.Interfaces
	for _, v := range nics {
		if v.MAC.Address == r.Mac {
			isFound = true
			break
		}
	}
	if !isFound {
		return errors.New("nic mac isn't exsist")
	}

	parts := []string{
		"virt-xml",
		r.Domain,
	}
	parts = append(parts, "--remove-device", "--network", "mac="+r.Mac)

	opt := &cmd.Option{}
	_, err = cmd.CmdCombinedBashWithCtx(context.TODO(), opt,
		strings.Join(parts, " "),
	)
	if err != nil {
		return err
	}

	return nil
}

type VmChangeCommonReq struct {
	Domain string `json:"domain" binding:"required"`
	//CpuMode  string      `json:"cpuMode" binding:"required"`
	//CpuModel string      `json:"cpuModel"`
	Memory uint64 `json:"memory" binding:"required"` // MB
	Vcpu   uint   `json:"vcpu" binding:"required"`
	//Boot     *BootOption `json:"boot" binding:"required"` // uefi,mbr
}

func VmChangeCommon(r *VmChangeCommonReq) error {
	dom, err := FindVm(r.Domain)
	if err != nil {
		return err
	}
	defer dom.Free()

	state, _, err := dom.GetState()
	if err != nil {
		return err
	}

	if state == libvirt.DOMAIN_RUNNING || state == libvirt.DOMAIN_PAUSED {
		return errors.New("vm is running or paused")
	}

	var tmp [][]string
	if r.Vcpu > 0 {
		tmp = append(tmp, []string{fmt.Sprintf("--vcpus %d", r.Vcpu)})
	}
	if r.Memory > 0 {
		tmp = append(tmp, []string{fmt.Sprintf("--memory memory=%d,maxmemory=%d", r.Memory, r.Memory)})
	}

	if len(tmp) == 0 {
		return errors.New("vm nothint to change")
	}

	for _, v := range tmp {
		if err = VmXmlEdit(r.Domain, v); err != nil {
			return err
		}
	}

	return nil
}

// virt-xml --edit only need one option
func VmXmlEdit(domain string, ls []string) error {
	parts := []string{
		"virt-xml",
		domain,
		"--edit",
	}

	parts = append(parts, ls...)

	opt := &cmd.Option{}
	_, err := cmd.CmdCombinedBashWithCtx(context.TODO(), opt,
		strings.Join(parts, " "),
	)
	if err != nil {
		return err
	}

	return nil
}
