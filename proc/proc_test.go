package proc

import (
	"os"
	"runtime"
	"strconv"
	"testing"
	// sigar "github.com/cloudfoundry/gosigar"
)

// func BenchmarkSigarMem(b *testing.B) {
// 	pid := os.Getpid()
// 	mem := sigar.ProcMem{}
// 	for i := 0; i < b.N; i++ {
// 		mem.Get(pid)
// 	}
// }

func BenchmarkRuntimeMem(b *testing.B) {
	var mem runtime.MemStats
	for i := 0; i < b.N; i++ {
		runtime.ReadMemStats(&mem)
	}
}

func BenchmarkMem(b *testing.B) {
	mem := &Mem{}
	pid := strconv.Itoa(os.Getpid())
	for i := 0; i < b.N; i++ {
		if err := mem.Get(pid); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRSS(b *testing.B) {
	pid := strconv.Itoa(os.Getpid())
	for i := 0; i < b.N; i++ {
		if _, err := GetRSS(pid); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcess(b *testing.B) {
	p, err := New(
		os.Getpid(),
		20,
		30,
		func(p float64, b uint64) bool { return true },
	)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Check()
	}
}
