package proc

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
)

// FreeFunc receives the amount of memory it should clear
// as a percentage and as bytes and should return if it was able to do so.
type FreeFunc func(pct float64, bytes uint64) bool

type Process struct {
	pid    string
	soft   uint64
	hard   uint64
	thresh uint64
	last   uint64
	freeer FreeFunc
}

func New(pid int, softMax, hardMax uint64, freeer FreeFunc) (*Process, error) {
	p := strconv.Itoa(pid)
	if hardMax <= softMax {
		return nil, errors.New("hardMax should be larger than softMax")
	}

	_, err := GetRSS(p)
	if err != nil {
		return nil, fmt.Errorf(
			"Seems parsing RSS from /proc/<pid>/statm is not supported: %s",
			err.Error(),
		)
	}

	threshold := hardMax - softMax
	if div := hardMax / 20; div < threshold {
		threshold = div
	}

	return &Process{
		p,
		softMax,
		hardMax,
		threshold,
		0,
		freeer,
	}, nil
}

func (p *Process) Check() {
	new, _ := GetRSS(p.pid)
	if new <= p.last {
		return
	}

	if new-p.last > p.thresh || new > p.soft {
		debug.FreeOSMemory()
		p.last, _ = GetRSS(p.pid)
		if p.last > p.soft && p.freeer(p.getPct(), p.last-p.soft) {
			debug.FreeOSMemory()
		}
	}
}

func (p *Process) getPct() float64 {
	return (1 - float64(p.soft)/float64(p.last))
}

func GetRSS(pid string) (uint64, error) {
	const split = ' '

	f, err := os.Open("/proc/" + pid + "/statm")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	r := bufio.NewReaderSize(f, 20)
	if _, err := r.ReadString(split); err != nil {
		return 0, err
	}

	v, err := r.ReadString(split)
	if err != nil {
		return 0, err
	}

	n, err := strconv.ParseUint(v[:len(v)-1], 10, 64)
	return n << 12, err
}

type Mem struct {
	Size  uint64
	Res   uint64
	Share uint64
}

func (m *Mem) Get(pid string) error {
	c, err := ioutil.ReadFile("/proc/" + pid + "/statm")
	if err != nil {
		return err
	}

	f := strings.Fields(string(c))
	if len(f) < 3 {
		return errors.New("Invalid statm format")
	}

	if m.Size, err = strconv.ParseUint(f[0], 10, 64); err != nil {
		return err
	}

	if m.Res, err = strconv.ParseUint(f[0], 10, 64); err != nil {
		return err
	}

	if m.Share, err = strconv.ParseUint(f[0], 10, 64); err != nil {
		return err
	}

	m.Size <<= 12
	m.Res <<= 12
	m.Share <<= 12
	return nil
}
