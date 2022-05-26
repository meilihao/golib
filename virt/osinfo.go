package virt

import (
	"bufio"
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/meilihao/golib/v2/cmd"
)

var (
	ErrNoOsinfo = errors.New("not found osinfo")
)

var (
	GlobalOsinfosLock sync.Mutex
	GlobalOsinfosErr  error
	GlobalOsinfos     []OsInfo
)

func init() {
	LoadOsinfos()
}

func LoadOsinfos() {
	GlobalOsinfosLock.Lock()
	defer GlobalOsinfosLock.Unlock()

	GlobalOsinfos, GlobalOsinfosErr = GetOsinfos()
	if GlobalOsinfosErr != nil {
		GlobalOsinfos = nil
	}
}

func ValidateOsinfo(family OsFamily, variant string) bool {
	tmp := string(family)

	for i := range GlobalOsinfos {
		if GlobalOsinfos[i].Family == tmp && GlobalOsinfos[i].ShortId == variant {
			return true
		}
	}

	return false
}

type OsInfo struct {
	ShortId string
	Name    string
	Version string
	Family  string
	Id      string
}

func GetOsinfos() (ls []OsInfo, err error) {
	c := "osinfo-query --fields=short-id,name,version,family,id os"

	opt := &cmd.Option{}
	out, err := cmd.CmdCombinedBashWithCtx(context.TODO(), opt, c)
	if err != nil {
		return
	}

	return ParseOsinfos(string(out))
}

func ParseOsinfos(raw string) (ls []OsInfo, err error) {
	var lines []string

	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}

	idx := -1
	for i := range lines {
		if strings.Contains(lines[i], "Short ID") {
			idx += 2
		}
	}
	if idx == -1 {
		err = ErrNoOsinfo
		return
	}

	lines = lines[idx:]
	ls = make([]OsInfo, 0, len(lines))
	var tmp []string
	for i := range lines {
		tmp = strings.Split(lines[i], "|")
		if len(tmp) != 5 {
			continue
		}
		for j := range tmp {
			tmp[j] = strings.TrimSpace(tmp[j])
		}

		ls = append(ls, OsInfo{
			ShortId: tmp[0],
			Name:    tmp[1],
			Version: tmp[2],
			Family:  tmp[3],
			Id:      tmp[4],
		})
	}

	return
}
