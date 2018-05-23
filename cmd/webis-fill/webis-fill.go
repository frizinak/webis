package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/frizinak/webis/cmd"
)

type entry struct {
	key    string
	tags   []string
	data   []byte
	ttl    string
	delete bool
}

func main() {
	data := flag.String("d", "", "Data")
	file := flag.String("f", "", "File")
	key := flag.String("k", "", "Key prefix")
	amount := flag.Uint("n", 2000, "Amount of entries to make")
	ns := flag.String("ns", "", "Namespace")
	delete := flag.Bool("D", false, "Delete afterwards")
	workers := flag.Int("J", 8, "Workers")

	host := flag.String("u", "", "Host")
	flag.Parse()

	if *workers < 1 {
		*workers = 1
	}

	input := []byte(*data)
	if *file != "" {
		raw, err := ioutil.ReadFile(*file)
		if err != nil {
			panic(err)
		}

		input = raw
	}

	if !strings.HasPrefix(*host, "http://") && !strings.HasPrefix(*host, "https://") {
		*host = "http://" + *host
	}
	ep, err := url.Parse(*host)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Invalid host and/or port")
		os.Exit(1)
	}
	cli := cmd.NewCLI(*ns, ep.String())

	var wg sync.WaitGroup
	work := make(chan *entry, *workers)

	del := func(w *entry) (int, []byte, error) { return cli.Del(w.key, nil) }
	set := func(w *entry) (int, []byte, error) { return cli.Set(w.key, w.tags, bytes.NewReader(w.data), w.ttl) }

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			for w := range work {
				cb := set
				if w.delete {
					cb = del
				}
				code, d, err := cb(w)
				if err != nil {
					panic(err)
				}
				if code != 201 && code != 200 {
					panic(string(d))
				}
			}
			wg.Done()
		}()
	}

	for i := uint(0); i < *amount; i++ {
		work <- &entry{*key + "fill-" + strconv.Itoa(int(i)), nil, input, "", false}
	}
	if *delete {
		for i := uint(0); i < *amount; i++ {
			work <- &entry{*key + "fill-" + strconv.Itoa(int(i)), nil, nil, "", true}
		}
	}

	close(work)
	wg.Wait()
}
