package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/frizinak/webis/cache"
	"github.com/frizinak/webis/server"
)

func main() {
	addr := flag.String("u", "localhost:3200", "Interface:port to listen on")
	hardMax := flag.Int64("hm", 512, "Hard memory limit in MiB")
	softMax := flag.Int64("sm", 350, "Soft memory limit in MiB")
	bodyMax := flag.Int64("b", 2048, "Post body size limit in KiB")
	verbose := flag.Bool("v", false, "Verbose")
	flag.Parse()

	if *softMax > *hardMax {
		*softMax = *hardMax
	}

	softMaxMem := uint64(*softMax * 1024 * 1024)
	hardMaxMem := uint64(*hardMax * 1024 * 1024)

	logger := log.New(os.Stderr, "", log.LstdFlags)
	cache := cache.New()
	purgeMem := make(chan struct{}, 1)

	go func() {
		for {
			cache.DelExpired()
			cache.Clean()
			time.Sleep(time.Second * 10)
		}
	}()

	go func() {
		var mem runtime.MemStats
		for {
			runtime.ReadMemStats(&mem)
			freeOS := func() (freeOS bool) {
				if mem.Alloc < softMaxMem {
					return
				}

				freeOS = mem.Alloc > hardMaxMem
				if *verbose {
					logger.Printf("OOM: looks like we're out of memory, forcing GC")
				}
				runtime.GC()
				runtime.ReadMemStats(&mem)
				if mem.Alloc < softMaxMem { //uint64(float64(softMaxMem)*0.95) {
					if *verbose {
						logger.Printf("OOM: seems we're ok: %d", mem.Alloc/1024)
					}
					return
				}

				fmaxMem := float64(softMaxMem)
				falloc := float64(mem.Alloc)
				pct := (1 - fmaxMem/falloc) + 0.02
				clears := int(float64(cache.Len()) * pct)
				if clears == 0 {
					clears = 1
				}

				logger.Printf(
					"OOM: (%.0f > %.0f) clearing %d random keys",
					falloc/1024,
					fmaxMem/1024,
					clears,
				)

				cache.DelRand(clears)
				if !freeOS {
					runtime.GC()
				}

				logger.Println("OOM: done")
				return
			}()

			if freeOS {
				purgeMem <- struct{}{}
			}

			time.Sleep(time.Millisecond * 100)
		}
	}()

	go func() {
		for {
			select {
			case <-purgeMem:
				logger.Printf("OOM: Freeing OS memory")
				debug.FreeOSMemory()
			case <-time.After(time.Minute * 2):
				debug.FreeOSMemory()
			}
		}
	}()

	serverLogger := log.New(ioutil.Discard, "", 0)
	if *verbose {
		serverLogger = logger
	}

	logger.Fatal(
		server.New(*addr, serverLogger, cache, *bodyMax*1024).Start(),
	)
}
