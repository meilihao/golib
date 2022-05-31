package virt

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/meilihao/golib/v2/misc"
)

// from webvirtcloud:is_supports_virtio, now
// virtinst:_supports_virtio
func IsSupportsVirtio(arch, machine string) bool {
	if misc.IsInStrings(arch, []string{"x86_64", "i686"}) {
		return true
	}

	if misc.IsInStrings(arch, []string{"aarch64"}) && misc.IsInStrings(machine, []string{"virt"}) {
		return true
	}

	return false
}

func GenerateRandomMac() string {
	a, b, c := rand.Int31n(255), rand.Int31n(255), rand.Int31n(255)
	return fmt.Sprintf("52:54:00:%02x:%02x:%02x", a, b, c)
}

func IsValidDiskName(bus string, name string) bool {
	switch bus {
	case BusIde:
		bus = "hd"
	case BusSata, BusScsi:
		bus = "sd"
	default:
		bus = "vd"
	}

	return bus != "" && strings.HasPrefix(name, bus)
}

type DiskFromNumber struct {
	Prefix string
	Start  uint
}

// start from 1...
func NewDiskFromNumber(bus string, start uint) *DiskFromNumber {
	switch bus {
	case BusIde:
		bus = "hd"
	case BusSata, BusScsi:
		bus = "sd"
	default:
		bus = "vd"
	}

	return &DiskFromNumber{
		Prefix: bus,
		Start:  start,
	}
}

func (n *DiskFromNumber) Generate() string {
	tmps := []string{}
	number, d := n.Start, uint(0)
	for number > 0 {
		number, d = divMod(number, 26)
		tmps = append(tmps, fmt.Sprintf("%c", d-1+97))
	}

	n.Start += 1

	// reverse tmps
	for i, j := 0, len(tmps)-1; i < j; i, j = i+1, j-1 {
		tmps[i], tmps[j] = tmps[j], tmps[i]
	}
	return n.Prefix + strings.Join(tmps, "")
}

func NewDiskFromNumberByName(name string) *DiskFromNumber {
	if !strings.HasPrefix(name, "hd") && !strings.HasPrefix(name, "sd") && !strings.HasPrefix(name, "vd") {
		return nil
	}

	bus := name[:2]
	name = name[2:]

	name = misc.Reverse(name)
	var number uint
	for i, letter := range []byte(name) {
		number += (uint(letter) - uint('a') + 1) * uint(math.Pow(26, float64(i)))
	}

	return &DiskFromNumber{
		Prefix: bus,
		Start:  number + 1,
	}
}

func divMod(x, y uint) (a uint, b uint) {
	a, b = x/y, x%y
	if b == 0 {
		return a - 1, b + 26
	}
	return a, b
}
