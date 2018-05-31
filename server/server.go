package server

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/frizinak/webis/cache"
)

const (
	HeaderKey  = "X-Key"
	HeaderTags = "X-Tags"
	HeaderNS   = "X-Namespace"
	HeaderTTL  = "X-TTL"
)

var tooLarge = errors.New("Too large")
var zeroRune rune = 0
var zero = string([]byte{byte(zeroRune)})

type Server struct {
	s           *http.Server
	c           *cache.Cache
	l           *log.Logger
	maxBodySize int
}

func (s *Server) handleList(
	w http.ResponseWriter,
	key cache.Key,
	tags []cache.Tag,
	r *http.Request,
) {
	if key == "" && len(tags) == 0 {
		w.WriteHeader(http.StatusNotAcceptable)
		fmt.Fprintf(w, "No tags or key provided")
		return
	}

	keys := true
	input := string(key)
	if input == "" {
		keys = false
		input = string(tags[0])
	}

	scan := scanner(
		input,
		func(k string) {
			w.Write([]byte(cleanDescriptor(k)))
			w.Write([]byte{10})
		},
	)

	w.WriteHeader(http.StatusOK)
	if keys {
		s.c.IterateKeys(
			func(k cache.Key) bool { scan(string(k)); return true },
		)
		return
	}

	s.c.IterateTags(
		func(k cache.Tag) bool { scan(string(k)); return true },
	)
}

func (s *Server) handleSet(
	w http.ResponseWriter,
	key cache.Key,
	tags []cache.Tag,
	r *http.Request,
) {
	if key == "" {
		w.WriteHeader(http.StatusNotAcceptable)
		fmt.Fprintf(w, "Invalid key")
		return
	}

	ttl, err := headerTTL(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusNotAcceptable)
		fmt.Fprintf(w, "Invalid ttl")
		return
	}

	reader := &limitReader{reader: r.Body, max: s.maxBodySize}
	data, err := ioutil.ReadAll(reader)
	r.Body.Close()
	if err != nil {
		if err == tooLarge {
			w.WriteHeader(http.StatusNotAcceptable)
			fmt.Fprintf(w, "Request body too large. Max %d KiB", s.maxBodySize/1024)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.c.Set(key, tags, string(data), time.Now().Add(ttl))

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "OK")
	s.l.Printf("Create %s\t%v\t%dkB", key, tags, len(data)/1024)
}

func (s *Server) handleGet(
	w http.ResponseWriter,
	key cache.Key,
	tags []cache.Tag,
	r *http.Request,
) {
	if key == "" && len(tags) == 0 {
		w.WriteHeader(http.StatusNotAcceptable)
		fmt.Fprintf(w, "No tags or key provided")
		return
	}

	if key != "" {
		v, ok := s.c.Get(key)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Not found")
			return
		}
		w.WriteHeader(http.StatusOK)

		if _, err := strings.NewReader(v).WriteTo(w); err != nil {
			s.l.Println(err)
		}
		return
	}

	v := s.c.GetTagKeys(tags[0])
	if v == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Not found")
		return
	}

	w.WriteHeader(http.StatusOK)
	for i := range v {
		w.Write([]byte(cleanDescriptor(string(v[i]))))
		w.Write([]byte{10})
	}
}

func (s *Server) handleDel(
	w http.ResponseWriter,
	key cache.Key,
	tags []cache.Tag,
	r *http.Request,
) {
	if key != "" {
		s.c.Del(key)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
		s.l.Printf("Delete %s", key)
		return
	}

	if len(tags) != 0 {
		for i := range tags {
			s.c.DelByTag(tags[i])
			s.l.Printf("Delete tag %s", tags[i])
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
		return
	}

	w.WriteHeader(http.StatusNotAcceptable)
	fmt.Fprintf(w, "No tags or key provided")
}

func (s *Server) handlePurge(
	w http.ResponseWriter,
	key cache.Key,
	tags []cache.Tag,
	r *http.Request,
) {
	if key != "" {
		w.WriteHeader(http.StatusNotAcceptable)
		fmt.Fprintf(w, "Providing a key makes little sense")
		return
	}

	key = headerKeyPrefix(r.Header)
	s.l.Printf("Purge NS %s", key)
	s.c.DelByPrefix(key)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func (s *Server) handlePurgeAll(
	w http.ResponseWriter,
	key cache.Key,
	tags []cache.Tag,
	r *http.Request,
) {
	s.l.Printf("Purge ALL")
	s.c.DelAll()
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func (s *Server) req(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	key := headerKey(r.Header)
	tags := headerTags(r.Header)
	var handler func(
		w http.ResponseWriter,
		key cache.Key,
		tags []cache.Tag,
		r *http.Request,
	) = nil

	switch {
	case path == "set" && r.Method == "POST":
		handler = s.handleSet
	case path == "get" && r.Method == "GET":
		handler = s.handleGet
	case path == "list" && r.Method == "GET":
		handler = s.handleList
	case path == "del" && r.Method == "POST":
		handler = s.handleDel
	case path == "purge" && r.Method == "POST":
		handler = s.handlePurge
	case path == "purge-all" && r.Method == "POST":
		handler = s.handlePurgeAll
	}

	if handler != nil {
		handler(w, key, tags, r)
		return
	}

	w.WriteHeader(http.StatusNotAcceptable)
}

func (s *Server) Start() error {
	return s.s.ListenAndServe()
}

func New(
	addr string,
	l *log.Logger,
	c *cache.Cache,
	maxBodySize int,
	readTimeout,
	readHeaderTimeout time.Duration,
) *Server {
	s := &http.Server{
		Addr:              addr,
		MaxHeaderBytes:    1024 * 80,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	server := &Server{s, c, l, maxBodySize}
	mux := http.NewServeMux()
	mux.HandleFunc("/", server.req)
	s.Handler = mux
	return server
}

type limitReader struct {
	reader io.Reader
	max    int
	read   int
	n      bool
}

func (l *limitReader) Read(b []byte) (int, error) {
	if l.n && len(b) > 0 {
		return 0, tooLarge
	}

	if len(b)+l.read > l.max {
		b = b[:l.max-l.read]
	}

	n, err := l.reader.Read(b)
	l.read += n
	if l.read >= l.max {
		l.n = true
	}

	return n, err
}

func tags(t []string) []cache.Tag {
	ts := make([]cache.Tag, len(t))
	for i := range t {
		ts[i] = cache.Tag(t[i])
	}
	return ts
}

func cleanDescriptor(i string) string {
	for j, n := range i {
		if n == zeroRune && len(i) > j+1 {
			return i[j+1:]
		}
	}
	return i

	r := strings.SplitN(i, zero, 2)
	if len(r) == 2 {
		return r[1]
	}
	return r[0]
}

func headerKey(h http.Header) cache.Key {
	k := h.Get(HeaderKey)
	if k == "" {
		return ""
	}

	return cache.Key(
		strings.Join([]string{h.Get(HeaderNS), h.Get(HeaderKey)}, zero),
	)
}

func headerKeyPrefix(h http.Header) cache.Key {
	return cache.Key(h.Get(HeaderNS) + zero)
}

func headerTags(h http.Header) []cache.Tag {
	t := h[HeaderTags]
	ts := make([]cache.Tag, len(t))
	for i := range t {
		ts[i] = cache.Tag(strings.Join([]string{h.Get(HeaderNS), t[i]}, zero))
	}
	return ts
}

func headerTTL(h http.Header) (time.Duration, error) {
	v := h[HeaderTTL]
	if len(v) == 0 {
		v = []string{h.Get(HeaderTTL)}
	}

	if len(v) == 0 || v[0] == "" {
		return math.MaxInt64, nil
	}

	i, err := strconv.ParseInt(v[0], 10, 32)
	if err != nil {
		return 0, err
	}

	return time.Duration(i) * time.Second, nil
}

func scanner(input string, match func(string)) func(string) {
	_ps := strings.Split(input, "*")
	return func(k string) {
		ps := _ps[:]
		key := k
		if ps[0] != "" {
			if !strings.HasPrefix(key, ps[0]) || (len(ps) == 1 && key != ps[0]) {
				return
			}
			key = key[len(ps[0]):]
			ps = ps[1:]
		}

		if len(ps) > 0 && ps[len(ps)-1] != "" {
			if !strings.HasSuffix(key, ps[len(ps)-1]) {
				return
			}
			key = key[:len(ps[len(ps)-1])]
			ps = ps[:len(ps)-1]
		}

		for _, p := range ps {
			i := strings.Index(key, p)
			if i == -1 {
				return
			}
			key = key[i+len(p):]
		}

		match(k)
		return
	}
}
