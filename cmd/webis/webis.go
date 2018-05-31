package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"runtime/debug"
	"time"

	"github.com/frizinak/webis/cache"
	"github.com/frizinak/webis/proc"
	"github.com/frizinak/webis/server"
)

func main() {
	addr := flag.String("u", "localhost:3200", "Interface:port to listen on")
	max := flag.Uint64("m", 512, "Memory limit in MiB")
	bodyMax := flag.Int("b", 2048, "Post body size limit in KiB")
	verbose := flag.Bool("v", false, "Verbose")
	flag.Parse()

	hardMaxMem := *max * 1024 * 1024
	softMaxMem := uint64(0.95 * float64(hardMaxMem))

	logger := log.New(os.Stderr, "", log.LstdFlags)
	cache := cache.New()

	debug.SetGCPercent(10)
	p, err := proc.New(
		os.Getpid(),
		softMaxMem,
		hardMaxMem,
		func(pct float64, b uint64) bool {
			clears := int(float64(cache.Len()) * (pct + 0.02))
			logger.Printf(
				"OOM: clearing %d random keys",
				clears,
			)
			cache.DelRand(clears)
			return true
		},
	)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			cache.DelExpired()
			cache.Clean()
			time.Sleep(time.Second * 10)
		}
	}()

	go func() {
		for {
			time.Sleep(time.Millisecond * 50)
			p.Check()
		}
	}()

	serverLogger := log.New(ioutil.Discard, "", 0)
	if *verbose {
		serverLogger = logger
	}

	logger.Println("Starting")
	logger.Fatal(
		server.New(
			*addr,
			serverLogger,
			cache,
			*bodyMax*1024,
			time.Second*5,
			time.Second,
		).Start(),
	)
}
