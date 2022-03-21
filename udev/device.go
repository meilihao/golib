package udev

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/meilihao/golib/v2/file"
)

var (
	ErrDeviceNotFound = errors.New("DeviceNotFound")
)

type SdDevice struct {
	Parent *SdDevice

	Name            string
	Devpath         string
	Envs            map[string]string
	Attrs           map[string]string
	Tags            []string
	UsecInitialized string // 当设备在 udevd 中注册时, USEC_INITIALIZED 设置为单调递增的系统时钟的值. `systemd/src/libsystemd/sd-device/device-private.c#device_ensure_usec_initialized`

	Devnum     string
	Subsystem  string
	IdFilename string

	_set_parent bool
}

func (d *SdDevice) GetName() string {
	if d.Name != "" {
		return d.Name
	}

	d.Name = filepath.Base(d.Devpath)
	return d.Name
}

func (d *SdDevice) GetParent() (*SdDevice, error) {
	if d._set_parent {
		return d.Parent, nil
	}

	if err := DeviceNewFromChild(d); err != nil {
		return nil, err
	}

	d._set_parent = true
	return d.Parent, nil
}

func DeviceNewFromChild(child *SdDevice) error {
	d := child.Devpath

	for {
		d = filepath.Dir(d)
		if !file.IsFile(filepath.Join(d, "/uevent")) {
			break
		}

		parent := &SdDevice{
			Envs:  map[string]string{},
			Attrs: map[string]string{},
			Tags:  make([]string, 0, 1),
		}

		if err := DeviceSetSyspath(parent, d, false); err != nil {
			return err
		}
		child.Parent = parent

		break
	}

	return nil
}

func (d *SdDevice) GetSubsystem() string {
	if d.Subsystem != "" {
		return d.Subsystem
	}

	if d.Envs["SUBSYSTEM"] != "" {
		d.Subsystem = d.Envs["SUBSYSTEM"]
		return d.Subsystem
	}

	tmp, _ := os.Readlink(filepath.Join(d.Devpath, "subsystem"))
	d.Subsystem = filepath.Base(tmp)
	return d.Subsystem
}

func (d *SdDevice) GetIdFilename() string {
	if d.IdFilename != "" {
		return d.IdFilename
	}

	tmp, _ := strconv.Atoi(d.Envs["MAJOR"])
	if tmp > 0 {
		if d.GetSubsystem() == "block" {
			d.IdFilename = fmt.Sprintf("%s%s", "b", d.Devnum)
		} else {
			d.IdFilename = fmt.Sprintf("%s%s", "c", d.Devnum)
		}

		return d.IdFilename
	}

	tmp, _ = strconv.Atoi(d.Envs["IFINDEX"]) // net device
	if tmp > 0 {
		d.IdFilename = fmt.Sprintf("%s%d", "n", tmp)
		return d.IdFilename
	}

	d.IdFilename = fmt.Sprintf("+%s:%s", d.Subsystem, filepath.Base(d.Devpath))
	return d.IdFilename
}

type UdevDevice struct {
	Parent *UdevDevice
	Device *SdDevice
}

// udev_device_new_from_subsystem_sysname
func FromName(subSystem, sysname string) (*UdevDevice, error) {
	device, err := SdDeviceNewFromSubsystemSysname(subSystem, sysname)
	if err != nil {
		return nil, err
	}

	return &UdevDevice{
		Device: device,
	}, nil
}

// sd_device_new_from_subsystem_sysname
func SdDeviceNewFromSubsystemSysname(subSystem, sysname string) (ret *SdDevice, err error) {
	sysname = strings.ReplaceAll(sysname, "/", "!")

	for _, v := range []string{"/sys/subsystem/", "/sys/bus/"} {
		if ret, err = DeviceStrjoinNew(v, subSystem, "/devices/", sysname); err == nil && ret != nil {
			return
		}
	}

	if ret, err = DeviceStrjoinNew("/sys/class", subSystem, "/", sysname); err == nil && ret != nil {
		return
	}

	return nil, ErrDeviceNotFound
}

func DeviceStrjoinNew(a, b, c, d string) (*SdDevice, error) {
	p := filepath.Join(a, b, c, d)
	if !file.IsExist(p) {
		return nil, nil
	}

	return SdDeviceNewFromSyspath(p)
}

// sd_device_new_from_syspath
func SdDeviceNewFromSyspath(syspath string) (r *SdDevice, err error) {
	r = &SdDevice{
		Envs:  map[string]string{},
		Attrs: map[string]string{},
		Tags:  make([]string, 0, 1),
	}

	err = DeviceSetSyspath(r, syspath, true)
	return
}

func DeviceSetSyspath(device *SdDevice, syspath string, verify bool) (err error) {
	if !strings.HasPrefix(syspath, "/sys/") {
		return fmt.Errorf("sd-device: syspath '%s' is not a subdirectory of /sys", syspath)
	}

	if verify {
		tmp, _ := os.Readlink(syspath)
		if tmp == "" {
			return fmt.Errorf("sd-device: could not get target of '%s'", syspath)
		}

		syspath, _ = filepath.Abs(filepath.Join(filepath.Dir(syspath), tmp))
		if !strings.HasPrefix(syspath, "/sys/devices/") {
			return fmt.Errorf("sd-device: could not canonicalize '%s'", syspath)
		}

		if !file.IsFile(filepath.Join(syspath, "/uevent")) {
			return fmt.Errorf("sd-device: %s does not have an uevent file", syspath)
		}
	}
	device.Devpath = syspath
	ReadAttrs(device)

	return ReadUevent(device)
}

func ReadAttrs(device *SdDevice) error {
	files, err := os.ReadDir(device.Devpath)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() || f.Name() == "uevent" || f.Name() == "descriptors" {
			continue
		}

		device.Attrs[f.Name()] = file.FileValue(filepath.Join(device.Devpath, f.Name()))
	}

	return nil
}

// https://github.com/citilinkru/libudev
func ReadUevent(device *SdDevice) error {
	data, _ := os.ReadFile(filepath.Join(device.Devpath, "/uevent"))
	buf := bufio.NewScanner(bytes.NewBuffer(data))

	var line string
	for buf.Scan() {
		line = buf.Text()
		field := strings.SplitN(line, "=", 2)
		if len(field) != 2 {
			continue
		}

		device.Envs[field[0]] = field[1]

	}
	device.Devnum = file.FileValue(filepath.Join(device.Devpath, "dev"))

	udevDataPath := "/run/udev/data/" + device.GetIdFilename()
	return ReadUdevInfo(udevDataPath, device)
}

func ReadUdevInfo(udevDataPath string, d *SdDevice) error {
	data, _ := os.ReadFile(udevDataPath)
	buf := bufio.NewScanner(bytes.NewBuffer(data))

	var line string
	for buf.Scan() {
		line = buf.Text()
		groups := strings.SplitN(line, ":", 2)
		if len(groups) != 2 {
			continue
		}

		if groups[0] == "I" {
			d.UsecInitialized = groups[1]
			continue
		}

		if groups[0] == "G" {
			d.Tags = append(d.Tags, groups[1])
			continue
		}

		if groups[0] == "E" {
			fields := strings.SplitN(groups[1], "=", 2)
			if len(fields) != 2 {
				continue
			}

			d.Envs[fields[0]] = fields[1]
		}
	}

	return nil
}

type FilterFn func(p string) bool

func ListDevices(subsystem string, fn FilterFn) (devices []*SdDevice, err error) {
	switch subsystem {
	case "block":
		return ListBlockDevices(fn)
	default:
		err = fmt.Errorf("no subsystem impl: %s", subsystem)
	}
	return
}

func ListBlockDevices(fn FilterFn) (devices []*SdDevice, err error) {
	ls, err := filepath.Glob("/sys/class/block/*")
	if err != nil {
		return
	}

	devices = make([]*SdDevice, 0)
	for _, p := range ls {
		if !fn(p) {
			continue
		}

		r := &SdDevice{
			Envs:  map[string]string{},
			Attrs: map[string]string{},
			Tags:  make([]string, 0, 1),
		}

		if err = DeviceSetSyspath(r, p, false); err != nil {
			return nil, err
		}

		devices = append(devices, r)
	}

	return
}

func WithFilterDevtype(name string) func(p string) bool {
	return func(p string) bool {
		data := file.FileValue(filepath.Join(p, "uevent"))
		if data == "" {
			return false
		}

		return strings.Contains(data, fmt.Sprintf("DEVTYPE=%s", name))
	}
}
