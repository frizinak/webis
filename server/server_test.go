package server

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	"github.com/frizinak/webis/cache"
)

func newServer() *Server {
	cache := cache.New()
	serverLogger := log.New(ioutil.Discard, "", 0)
	return New(":8080", serverLogger, cache, 10*1024*1024)
}

type responseWriter struct {
	header http.Header
	buf    *bytes.Buffer
	code   int
}

func (r *responseWriter) Header() http.Header {
	return r.header
}

func (r *responseWriter) Write(n []byte) (int, error) {
	return r.buf.Write(n)
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.code = statusCode
}

func makeReq(
	s *Server,
	method,
	path string,
	input []byte,
	ns,
	key,
	ttl string,
	tags []string,
) (code int, body []byte, err error) {
	res := &responseWriter{make(http.Header), bytes.NewBuffer(nil), 0}
	url := "http://localhost/" + path
	post := bytes.NewReader(input)
	var req *http.Request
	req, err = http.NewRequest(method, url, post)
	if err != nil {
		return
	}
	req.Header[HeaderKey] = []string{key}
	req.Header[HeaderNS] = []string{ns}
	req.Header[HeaderTags] = tags
	req.Header[HeaderTTL] = []string{ttl}

	s.req(res, req)
	code = res.code
	body = res.buf.Bytes()
	return
}

func testReq(t *testing.T, expectedCode, code int, body []byte, err error) {
	if err != nil {
		t.Fatal(err)
	}

	if code != expectedCode {
		t.Fatalf("Code: %d Resp: %s", code, string(body))
	}
}

func TestSimpleIntegrationGetSetDel(t *testing.T) {
	s := newServer()

	nss := []string{"", "ns"}
	for _, ns := range nss {
		// SET key [tag1, tag2]
		code, data, err := makeReq(s, "POST", "set", []byte("data"+ns), ns, "key", "100", []string{"tag1", "tag2"})
		testReq(t, http.StatusCreated, code, data, err)

		// SET key2 [tag2]
		code, data, err = makeReq(s, "POST", "set", []byte("data"+ns), ns, "key2", "100", []string{"tag2"})
		testReq(t, http.StatusCreated, code, data, err)
	}

	for _, ns := range nss {
		// GET key
		code, data, err := makeReq(s, "GET", "get", nil, ns, "key", "", nil)
		testReq(t, http.StatusOK, code, data, err)
		if string(data) != "data"+ns {
			t.Fatalf("Stored data does not match what was retrieved: %s", data)
		}

		// DEL tag1
		code, data, err = makeReq(s, "POST", "del", nil, ns, "", "", []string{"tag1"})
		testReq(t, http.StatusOK, code, data, err)

		// GET key
		code, data, err = makeReq(s, "GET", "get", nil, ns, "key", "", nil)
		testReq(t, http.StatusNotFound, code, data, err)

		// GET key2
		code, data, err = makeReq(s, "GET", "get", nil, ns, "key2", "", nil)
		testReq(t, http.StatusOK, code, data, err)

		// DEL key2
		code, data, err = makeReq(s, "POST", "del", nil, ns, "key2", "", nil)
		testReq(t, http.StatusOK, code, data, err)

		// GET key2
		code, data, err = makeReq(s, "GET", "get", nil, ns, "key2", "", nil)
		testReq(t, http.StatusNotFound, code, data, err)
	}
}

func TestWildcard(t *testing.T) {
	test := func(d map[string]int, input string) {
		s := scanner(input, func(k string) {
			d[k]--
			if d[k] < 0 {
				t.Errorf("%s should not match for %s", k, input)
			}
		})

		for i := range d {
			s(i)
			if d[i] > 0 {
				t.Errorf("%s should've matched for %s", i, input)
			}
		}
	}

	test(
		map[string]int{
			"inputlalatest":    1,
			"input lala test":  1,
			"sinputlalatest":   0,
			"sinput lala test": 0,
			"input lala tests": 0,
			"":                 0,
		},
		"input*test",
	)

	test(
		map[string]int{
			"inputlalatest":    1,
			"input lala test":  1,
			"sinputlalatest":   1,
			"sinput lala test": 1,
			"input lala tests": 1,
			"inputlalate":      0,
			"input lala te":    0,
			"":                 0,
		},
		"*input*test*",
	)
	test(
		map[string]int{
			"inputlalatest":    1,
			"input lala test":  1,
			"sinputlalatest":   1,
			"sinput lala test": 1,
			"input lala tests": 1,
			"inputlalate":      1,
			"input lala te":    1,
			"wefwf":            1,
			"":                 1,
		},
		"*",
	)

	test(
		map[string]int{
			"inputlalatest":    1,
			"input lala test":  0,
			"sinputlalatest":   0,
			"sinput lala test": 0,
			"input lala tests": 0,
			"inputlalate":      0,
			"input lala te":    0,
			"wefwf":            0,
			"":                 0,
		},
		"inputlalatest",
	)

	test(
		map[string]int{
			"prefix":   1,
			"prefixed": 0,
		},
		"prefix",
	)

	test(
		map[string]int{
			"suffix":    1,
			"abcsuffix": 0,
		},
		"suffix",
	)

	test(
		map[string]int{
			"prefix":   1,
			"prefixed": 1,
		},
		"prefix*",
	)

	test(
		map[string]int{
			"suffix":    1,
			"abcsuffix": 1,
		},
		"*suffix",
	)
}
