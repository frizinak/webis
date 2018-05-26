package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/frizinak/webis/cmd"
)

type list []string

func (l *list) String() string {
	return "List"
}

func (l *list) Set(v string) error {
	*l = append(*l, v)
	return nil
}

func main() {
	var tags list
	methodSet := flag.Bool("S", false, "SET")
	methodGet := flag.Bool("G", false, "GET")
	methodDel := flag.Bool("D", false, "DEL")
	methodList := flag.Bool("L", false, "LIST")
	methodPurge := flag.Bool("PURGE", false, "PURGE")
	methodPurgeAll := flag.Bool("PURGE-ALL", false, "PURGE")
	data := flag.String("d", "", "Data")
	file := flag.String("f", "", "File")
	key := flag.String("k", "", "Key")
	ns := flag.String("ns", "", "Namespace")
	ttl := flag.String("ttl", "", "ttl")
	flag.Var(&tags, "t", "Tags")

	host := flag.String("u", "localhost:3200", "Host")
	flag.Parse()

	var reader io.Reader
	reader = bytes.NewReader([]byte(*data))
	if *file != "" {
		f, err := os.Open(*file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer f.Close()
		reader = f
	}

	if !strings.HasPrefix(*host, "http://") && !strings.HasPrefix(*host, "https://") {
		*host = "http://" + *host
	}
	ep, err := url.Parse(*host)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Invalid host and/or port")
		os.Exit(1)
	}

	var was bool
	for _, b := range []bool{
		*methodSet,
		*methodGet,
		*methodDel,
		*methodList,
		*methodPurge,
		*methodPurgeAll,
	} {
		if !b {
			continue
		}
		if was {
			fmt.Fprintln(os.Stderr, "Multiple methods selected")
			os.Exit(1)
		}
		was = true
	}

	cli := cmd.NewCLI(*ns, ep.String())
	if *methodSet {
		if *key == "" {
			fmt.Fprintln(os.Stderr, "No key specified (-k)")
			os.Exit(1)
		}
	}

	switch {
	case *methodSet:
		cmd.Print(cli.Set(*key, tags, reader, *ttl))
	case *methodGet:
		cmd.PrintReader(cli.Get(*key, tags))
	case *methodDel:
		cmd.Print(cli.Del(*key, tags))
	case *methodList:
		cmd.PrintReader(cli.List(*key, tags))
	case *methodPurge:
		cmd.Print(cli.Purge())
	case *methodPurgeAll:
		cmd.Print(cli.PurgeAll())
	default:
		fmt.Fprintln(os.Stderr, "No method selected")
		flag.Usage()
		os.Exit(1)
	}
}
