package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/frizinak/webis/server"
)

type CLI struct {
	ns string
	ep string
	c  *http.Client
}

func NewCLI(ns, ep string) *CLI {
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 1000
	return &CLI{ns, ep, &http.Client{}}
}

func (c *CLI) Set(
	key string,
	tags []string,
	r io.Reader,
	ttl string,
) (int, []byte, error) {
	req, err := http.NewRequest("POST", c.ep+"/set", r)
	if err != nil {
		return 0, nil, err
	}

	req.Header[server.HeaderKey] = []string{key}
	req.Header[server.HeaderTags] = tags
	req.Header[server.HeaderNS] = []string{c.ns}
	req.Header[server.HeaderTTL] = []string{ttl}

	return c.do(req)
}

func (c *CLI) Get(key string, tags []string) (int, io.ReadCloser, error) {
	req, err := http.NewRequest("GET", c.ep+"/get", nil)
	if err != nil {
		return 0, nil, err
	}

	req.Header[server.HeaderKey] = []string{key}
	req.Header[server.HeaderTags] = tags
	req.Header[server.HeaderNS] = []string{c.ns}

	res, err := c.c.Do(req)
	code := 0
	var body io.ReadCloser
	if res != nil {
		code = res.StatusCode
		body = res.Body
	}

	return code, body, err
}

func (c *CLI) Del(key string, tags []string) (int, []byte, error) {
	req, err := http.NewRequest("POST", c.ep+"/del", nil)
	if err != nil {
		return 0, nil, err
	}

	req.Header[server.HeaderKey] = []string{key}
	req.Header[server.HeaderTags] = tags
	req.Header[server.HeaderNS] = []string{c.ns}

	return c.do(req)
}

func (c *CLI) Purge() (int, []byte, error) {
	req, err := http.NewRequest("POST", c.ep+"/purge", nil)
	if err != nil {
		return 0, nil, err
	}

	req.Header[server.HeaderNS] = []string{c.ns}

	return c.do(req)
}

func (c *CLI) PurgeAll() (int, []byte, error) {
	req, err := http.NewRequest("POST", c.ep+"/purge-all", nil)
	if err != nil {
		return 0, nil, err
	}

	return c.do(req)
}

func (c *CLI) List(key string, tags []string) (int, io.ReadCloser, error) {
	req, err := http.NewRequest("GET", c.ep+"/list", nil)
	if err != nil {
		return 0, nil, err
	}

	req.Header[server.HeaderKey] = []string{key}
	req.Header[server.HeaderTags] = tags
	req.Header[server.HeaderNS] = []string{c.ns}

	res, err := c.c.Do(req)
	code := 0
	var body io.ReadCloser
	if res != nil {
		code = res.StatusCode
		body = res.Body
	}

	return code, body, err
}

func (c *CLI) do(req *http.Request) (int, []byte, error) {
	res, err := c.c.Do(req)
	code := 0
	if res != nil {
		code = res.StatusCode
	}

	if res != nil && res.Body != nil {
		d, _ := ioutil.ReadAll(res.Body)
		res.Body.Close()
		return code, d, err
	}

	return code, nil, err
}

func Print(code int, data []byte, err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	fmt.Fprintf(os.Stderr, "[%d] %s\n", code, data)
}

func PrintReader(code int, r io.ReadCloser, err error) {
	if code == 200 && err == nil {
		defer r.Close()
		if _, err = io.Copy(os.Stdout, r); err == nil {
			return
		}
	}
	Print(code, nil, err)
}
