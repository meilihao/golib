package virt

import (
	"bufio"
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/hashicorp/go-version"
	"github.com/meilihao/golib/v2/cmd"
	"github.com/meilihao/golib/v2/misc"
)

var (
	ErrNoOsinfo = errors.New("not found osinfo")
)

var (
	GlobalOsinfosLock sync.Mutex
	GlobalOsinfosErr  error
	GlobalOsinfos     []OsInfo

	OsVersionFilters = map[string]func(in string) bool{
		"winnt": OsVersionFilterBtWinnt,
	}
)

func OsVersionFilterBtWinnt(in string) bool {
	v, _ := version.NewVersion(in)
	c, _ := version.NewConstraint(">= 5.0")

	if v == nil || c == nil {
		return false
	}

	return c.Check(v)
}

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
	ShortId string `json:"shortId"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Family  string `json:"family"`
	Id      string `json:"id"`
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

// winnt > = 5.0
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
	var f func(in string) bool
	for i := range lines {
		tmp = strings.Split(lines[i], "|")
		if len(tmp) != 5 {
			continue
		}
		for j := range tmp {
			tmp[j] = strings.TrimSpace(tmp[j])
		}

		if f = OsVersionFilters[tmp[3]]; f != nil && !f(tmp[2]) {
			continue
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

type OsinfoFilter struct {
	OsFamilies []string `json:"osFamilies"` // empty return all. common is "linux,winnt"
}

func GetOsinfosWithFilter(f *OsinfoFilter) (ls []OsInfo, err error) {
	if GlobalOsinfosErr != nil {
		return make([]OsInfo, 0), GlobalOsinfosErr
	}

	if len(f.OsFamilies) == 0 {
		return GlobalOsinfos, nil
	}

	ls = make([]OsInfo, 0)
	for _, v := range GlobalOsinfos {
		if misc.IsInStrings(v.Family, f.OsFamilies) {
			ls = append(ls, v)
		}
	}

	return ls, nil
}
