package cache

import (
	"sync"
	"time"
)

type Tag string
type Key string

type Cache struct {
	dsem sync.RWMutex
	tsem sync.RWMutex
	data map[Key]*entry
	tags map[Tag]*tags
}

func New() *Cache {
	var dsem sync.RWMutex
	var tsem sync.RWMutex
	return &Cache{
		dsem,
		tsem,
		make(map[Key]*entry, 100),
		make(map[Tag]*tags),
	}
}

// https://github.com/golang/go/issues/20135
// func (c *Cache) Recreate() {
// 	d := make(map[Key]*entry, c.Len())
// 	c.dsem.RLock()
// 	for i := range c.data {
// 		d[i] = c.data[i]
// 	}
// 	c.dsem.RUnlock()
//
// 	c.dsem.Lock()
// 	c.data = d
// 	c.dsem.Unlock()
//
// 	t := make(map[Tag]*tags, c.TagsLen())
// 	c.tsem.RLock()
// 	for i := range c.tags {
// 		t[i] = c.tags[i]
// 	}
// 	c.tsem.RUnlock()
//
// 	c.tsem.Lock()
// 	c.tags = t
// 	c.tsem.Unlock()
// }

func (c *Cache) Len() int {
	c.dsem.RLock()
	l := len(c.data)
	c.dsem.RUnlock()
	return l
}

func (c *Cache) TagsLen() int {
	c.tsem.RLock()
	l := len(c.tags)
	c.tsem.RUnlock()
	return l
}

func (c *Cache) Set(key Key, tags []Tag, value string, expires time.Time) {
	c.dsem.Lock()
	c.data[key] = &entry{value, expires}
	c.dsem.Unlock()

	for _, t := range tags {
		c.tsem.RLock()
		keys := c.tags[t]
		c.tsem.RUnlock()
		if keys == nil {
			c.tsem.Lock()
			c.tags[t] = newTags()
			c.tsem.Unlock()
		}
		c.tags[t].add(key)
	}
}

func (c *Cache) Get(key Key) (string, bool) {
	c.dsem.RLock()
	d := c.data[key]
	c.dsem.RUnlock()

	if d == nil {
		return "", false
	}

	if d.e.Before(time.Now()) {
		return "", false
	}

	return d.d, true
}

func (c *Cache) GetTagKeys(tag Tag) []Key {
	c.tsem.RLock()
	t := c.tags[tag]
	c.tsem.RUnlock()
	if t == nil {
		return nil
	}

	return t.get()

}

func (c *Cache) IterateKeys(cb func(Key) bool) {
	c.dsem.RLock()
	for i := range c.data {
		if !cb(i) {
			break
		}
	}
	c.dsem.RUnlock()
}

func (c *Cache) IterateTags(cb func(Tag) bool) {
	c.tsem.RLock()
	for i := range c.tags {
		if !cb(i) {
			break
		}
	}
	c.tsem.RUnlock()
}

func (c *Cache) Del(key Key) {
	c.dsem.Lock()
	c.del(key)
	c.dsem.Unlock()
}

func (c *Cache) DelByTag(tag Tag) {
	c.tsem.Lock()
	t := c.tags[tag]
	delete(c.tags, tag)
	c.tsem.Unlock()
	if t == nil {
		return
	}

	t.delFromCache(c)
}

func (c *Cache) DelByPrefix(prefix Key) {
	c.dsem.Lock()
	for i := range c.data {
		if len(i) >= len(prefix) && i[0:len(prefix)] == prefix {
			c.del(i)
		}
	}
	c.dsem.Unlock()
}

func (c *Cache) DelExpired() {
	c.delExpired(c.Len() / 100)
}

func (c *Cache) DelRand(n int) {
	if n <= 0 {
		return
	}

	c.dsem.Lock()
	for i := range c.data {
		c.del(i)
		if n--; n <= 0 {
			break
		}
	}

	c.dsem.Unlock()
}

func (c *Cache) DelAll() {
	data := make(map[Key]*entry)
	tags := make(map[Tag]*tags)

	c.dsem.Lock()
	c.tsem.Lock()
	c.data = data
	c.tags = tags
	c.dsem.Unlock()
	c.tsem.Unlock()
}

func (c *Cache) Clean() {
	size := c.TagsLen() / 10
	c.cleanTagKeys(size)
	c.cleanTags(size)
}

func (c *Cache) cleanTagKeys(scans int) {
	c.tsem.RLock()
	for i := range c.tags {
		c.tags[i].delNotInCache(c)
		if scans--; scans <= 0 {
			break
		}
	}
	c.tsem.RUnlock()
}

func (c *Cache) cleanTags(scans int) {
	clear := make([]Tag, 0)
	c.tsem.RLock()
	for i := range c.tags {
		if c.tags[i].IsEmpty() {
			clear = append(clear, i)
		}
		if scans--; scans <= 0 {
			break
		}
	}
	c.tsem.RUnlock()

	if len(clear) == 0 {
		return
	}

	c.tsem.Lock()
	for i := range clear {
		delete(c.tags, clear[i])
	}
	c.tsem.Unlock()
}

func (c *Cache) del(key Key) {
	delete(c.data, key)
}

func (c *Cache) delExpired(scans int) {
	clear := make([]Key, 0)
	now := time.Now()
	c.dsem.RLock()
	for i := range c.data {
		if c.data[i].e.Before(now) {
			clear = append(clear, i)
		}

		if scans--; scans <= 0 {
			break
		}
	}
	c.dsem.RUnlock()
	c.dsem.Lock()
	for i := range clear {
		c.del(clear[i])
	}

	c.dsem.Unlock()
}

type entry struct {
	d string
	e time.Time
}

type tags struct {
	sem sync.RWMutex
	t   map[Key]struct{}
}

func newTags() *tags {
	var sem sync.RWMutex
	return &tags{sem, make(map[Key]struct{}, 0)}
}

func (t *tags) get() []Key {
	t.sem.RLock()
	l := make([]Key, 0, len(t.t))
	for i := range t.t {
		l = append(l, i)
	}
	t.sem.RUnlock()
	return l
}

func (t *tags) add(key Key) {
	t.sem.Lock()
	t.t[key] = struct{}{}
	t.sem.Unlock()
}

func (t *tags) delFromCache(c *Cache) {
	t.sem.RLock()
	c.dsem.Lock()
	for i := range t.t {
		c.del(i)
	}
	c.dsem.Unlock()
	t.sem.RUnlock()
}

func (t *tags) delNotInCache(c *Cache) {
	clear := make([]Key, 0)
	t.sem.RLock()
	c.dsem.RLock()
	for i := range t.t {
		if _, ok := c.data[i]; !ok {
			clear = append(clear, i)
		}
	}
	c.dsem.RUnlock()
	t.sem.RUnlock()

	if len(clear) == 0 {
		return
	}

	t.sem.Lock()
	for i := range clear {
		delete(t.t, clear[i])
	}
	t.sem.Unlock()
}

func (t *tags) IsEmpty() bool {
	t.sem.RLock()
	ret := len(t.t) == 0
	t.sem.RUnlock()
	return ret
}
