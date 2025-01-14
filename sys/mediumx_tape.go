package sys

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/meilihao/golib/v2/cmd"
	"github.com/meilihao/golib/v2/file"
	"github.com/meilihao/golib/v2/log"
	"go.uber.org/zap"
)

var (
	refreshStatus atomic.Int32
)

const (
	TypeMediumx = "mediumx"
	TypeTape    = "tape"
)

type Tape struct {
	Bus      string `json:"bus"`
	Vendor   string `json:"vendor"`
	Model    string `json:"model"`
	Rev      string `json:"rev"`
	Device   string `json:"device"`
	Sg       string `json:"sg"`
	PathByid string `json:"path_byid"`
}

type Mediumx struct {
	Bus      string `json:"bus"`
	Vendor   string `json:"vendor"`
	Model    string `json:"model"`
	Rev      string `json:"rev"`
	Device   string `json:"device"`
	Sg       string `json:"sg"`
	PathByid string `json:"path_byid"`

	Tapes []*Tape `json:"tapes"`

	Target *TargetFrom `json:"target"`
}

const (
	ProtocolIscsi = "iscsi"
	ProtocolFc    = "fc"
)

type TargetFrom struct {
	Protocol   string `json:"protocol"`
	Target     string `json:"target"`
	User       string `json:"user"`
	Password   string `json:"password"`
	ServerIp   string `json:"server_ip"`
	ServerPort int    `json:"server_port"`
}

func RefreshFc() (s int32) {
	s = refreshStatus.Load()
	if s > 0 {
		return
	}

	refreshStatus.Add(1)
	defer refreshStatus.Add(-1)

	fcls, _ := filepath.Glob("/sys/class/fc_host/host*")
	if len(fcls) > 0 {
		data, err := cmd.CmdCombinedBash(nil, `echo "- - -" | tee -a /sys/class/scsi_host/*/scan`)
		if err != nil {
			log.Glog.Error("scan fc", zap.String("out", string(data)), zap.Error(err))
		}

		time.Sleep(3 * time.Second)
	}

	return
}

// 物理带库, 可能TypeTape在前, TypeMediumx在后
func GetMediumxs() ([]*Mediumx, error) {
	data, err := cmd.CmdCombinedBash(nil, "lsscsi -g")
	if err != nil {
		return nil, err
	}

	var tmp string
	var lines [][]string
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		tmp = strings.TrimSpace(sc.Text())
		if tmp == "" {
			continue
		}

		lines = append(lines, strings.Fields(tmp))
	}

	var idx int
	var num int
	var tmpMediumx *Mediumx
	ls := make([]*Mediumx, 0)
	m := make(map[string]*Mediumx, 3)
	for _, v := range lines {
		if len(v) < 7 {
			continue
		}
		num = len(v)

		if v[1] == TypeMediumx {
			tmpMediumx = &Mediumx{
				Bus:    strings.TrimPrefix(strings.TrimSuffix(v[0], "]"), "["),
				Rev:    v[num-3],
				Device: v[num-2],
				Sg:     v[num-1],
			}
			tmpMediumx.Vendor = file.FileValue(filepath.Join("/sys/bus/scsi/devices", tmpMediumx.Bus, "vendor"))
			tmpMediumx.Model = file.FileValue(filepath.Join("/sys/bus/scsi/devices", tmpMediumx.Bus, "model")) // may has space

			target, err := GetMediumxTarget(tmpMediumx.Sg)
			if err != nil {
				return nil, err
			}
			tmpMediumx.Target = target

			ls = append(ls, tmpMediumx)

			idx = strings.LastIndex(tmpMediumx.Bus, ":")
			m[tmpMediumx.Bus[:idx]] = tmpMediumx
		}
	}

	if len(ls) == 0 {
		return ls, nil
	}

	for _, v := range lines {
		if len(v) < 7 {
			continue
		}
		num = len(v)

		if v[1] == TypeTape {
			tp := &Tape{
				Bus:    strings.TrimPrefix(strings.TrimSuffix(v[0], "]"), "["),
				Rev:    v[num-3],
				Device: v[num-2],
				Sg:     v[num-1],
			}

			idx = strings.LastIndex(tp.Bus, ":")
			tmpMediumx = m[tp.Bus[:idx]]
			if tmpMediumx == nil {
				return nil, fmt.Errorf("found tape(%s) with no mediumx", v[0])
			}

			tp.Vendor = file.FileValue(filepath.Join("/sys/bus/scsi/devices", tp.Bus, "vendor"))
			tp.Model = file.FileValue(filepath.Join("/sys/bus/scsi/devices", tp.Bus, "model"))

			tmpMediumx.Tapes = append(tmpMediumx.Tapes, tp)
		}
	}

	byIds, err := TapeByIdPaths()
	if err != nil {
		return nil, err
	}

	for _, m := range ls {
		m.PathByid = byIds[filepath.Base(m.Sg)]
		if m.PathByid == "" {
			return nil, fmt.Errorf("mediumx(%s) no byid path", m.Bus)
		}

		for _, tp := range m.Tapes {
			tp.PathByid = byIds["n"+filepath.Base(tp.Device)]
			if tp.PathByid == "" {
				return nil, fmt.Errorf("tape(%s) no byid path", tp.Bus)
			}
		}
	}

	return ls, nil
}

func TapeByIdPaths() (map[string]string, error) {
	base := "/dev/tape/by-id"
	fs, err := ioutil.ReadDir(base)
	if err != nil {
		return nil, err
	}

	var tmp string
	m := make(map[string]string, len(fs))

	for _, f := range fs {
		if f.Mode()&os.ModeSymlink == 0 {
			continue
		}

		tmp, _ = os.Readlink(filepath.Join(base, f.Name()))
		tmp = filepath.Base(tmp)

		if strings.HasPrefix(tmp, "sg") {
			m[tmp] = filepath.Join(base, f.Name())
		} else if strings.HasPrefix(tmp, "nst") {
			m[tmp] = filepath.Join(base, f.Name())
		}
	}

	return m, nil
}

func GetMediumxTarget(dev string) (*TargetFrom, error) {
	data, _ := cmd.CmdCombinedBash(&cmd.Option{IgnoreErr: true}, fmt.Sprintf("udevadm info %s | grep 'E: ID_PATH='", dev))
	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return nil, nil
	}

	if strings.Contains(raw, "-iscsi-") {
		x := strings.Index(raw, "-iscsi-")
		y := strings.LastIndex(raw, "-lun-")
		if y == -1 {
			return nil, fmt.Errorf("no found target(%s)", dev)
		}

		return &TargetFrom{
			Protocol: ProtocolIscsi,
			Target:   raw[x+7 : y],
		}, nil
	}
	if strings.Contains(raw, "-fc-") {
		x := strings.Index(raw, "-fc-")
		y := strings.LastIndex(raw, "-lun-")
		if y == -1 {
			return nil, fmt.Errorf("no found target(%s)", dev)
		}

		return &TargetFrom{
			Protocol: ProtocolFc,
			Target:   raw[x+4 : y],
		}, nil
	}

	return nil, nil
}
