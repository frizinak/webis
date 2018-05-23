package cache

import (
	"crypto/rand"
	"io"
	mrand "math/rand"
	"strconv"
	"testing"
	"time"
)

func newCache() *Cache {
	return New()
}

func newData(random bool) string {
	data := make([]byte, 1e2)
	if !random {
		return string(data)
	}

	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		panic(err)
	}

	return string(data)
}

func BenchmarkSet(b *testing.B) {
	data := newData(true)
	expires := time.Now().Add(time.Second * 100)
	cache := newCache()
	tags := []Tag{"wopla", "dopla"}

	k := Key("lala")
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Set(k, tags, data, expires)
		}
	})
}

func BenchmarkSetAlloc(b *testing.B) {
	expires := time.Now().Add(time.Second * 100)
	cache := newCache()
	tags := []Tag{"wopla", "dopla"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			k := Key(strconv.Itoa(mrand.Intn(1e9)))
			cache.Set(k, tags, newData(false), expires)
		}
	})
}

func BenchmarkGet(b *testing.B) {
	data := newData(true)
	cache := newCache()
	expires := time.Now().Add(time.Second * 100)
	tags := []Tag{"wopla", "dopla"}
	var max int = 1e3
	for i := 0; i < max; i++ {
		cache.Set(
			Key(strconv.Itoa(i)),
			tags,
			data,
			expires,
		)
	}

	k := Key(strconv.Itoa(mrand.Intn(max)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, ok := cache.Get(k); !ok {
				b.Error("OOPS")
			}
		}
	})
}

func BenchmarkDelByTag(b *testing.B) {
	data := newData(true)
	cache := newCache()
	expires := time.Now().Add(time.Second * 100)
	var max int = 1e3
	var tagsets int = 1e2
	for j := 0; j < tagsets; j++ {
		tags := []Tag{"wopla" + Tag(strconv.Itoa(j)), "dopla"}
		for i := 0; i < max; i++ {
			cache.Set(
				Key(strconv.Itoa(i)+":"+strconv.Itoa(j)),
				tags,
				data,
				expires,
			)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			i++
			cache.DelByTag("wopla" + Tag(strconv.Itoa(i%tagsets)))
		}
	})
}

func BenchmarkDelExpired(b *testing.B) {
	data := newData(true)
	cache := newCache()
	expires := time.Now().Add(-time.Second)
	var max int = 1e4
	tags := []Tag{"wopla", "dopla"}
	for i := 0; i < max; i++ {
		cache.Set(
			Key(strconv.Itoa(i)),
			tags,
			data,
			expires,
		)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.delExpired(1e2)
		}
	})
}

func BenchmarkClean(b *testing.B) {
	data := newData(true)
	cache := newCache()
	expires := time.Now().Add(time.Second * 100)
	var max int = 1e2
	var tagsets int = 1e4
	for j := 0; j < tagsets; j++ {
		tags := []Tag{"wopla" + Tag(strconv.Itoa(j)), "dopla"}
		for i := 0; i < max; i++ {
			k := Key(strconv.Itoa(i) + ":" + strconv.Itoa(j))
			cache.Set(k, tags, data, expires)
			cache.Del(k)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.cleanTagKeys(10)
			cache.cleanTags(10)
		}
	})
}

// func BenchmarkRecreate(b *testing.B) {
// 	data := newData(true)
// 	cache := newCache()
// 	expires := time.Now().Add(-time.Second)
// 	var max int = 1e4
// 	tags := []Tag{"wopla", "dopla"}
// 	for i := 0; i < max; i++ {
// 		cache.Set(
// 			Key(strconv.Itoa(i)),
// 			tags,
// 			data,
// 			expires,
// 		)
// 	}
//
// 	b.ResetTimer()
// 	b.RunParallel(func(pb *testing.PB) {
// 		for pb.Next() {
// 			cache.Recreate()
// 		}
// 	})
// }

func BenchmarkDelByPrefix(b *testing.B) {
	data := newData(true)
	cache := newCache()
	expires := time.Now().Add(time.Second * 100)

	var prefix Key = "ze-prefix-"
	var max int = 1e3
	for i := 0; i < b.N; i++ {
		p := prefix + Key(strconv.Itoa(i))
		for j := 0; j < max; j++ {
			cache.Set(
				p+Key(strconv.Itoa(j)),
				nil,
				data,
				expires,
			)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := prefix + Key(strconv.Itoa(i))
		cache.DelByPrefix(p)
	}
}

func TestDelGet(t *testing.T) {
	cache := newCache()
	data := "data"
	var key Key = "key"
	cache.Set(key, []Tag{}, data, time.Now().Add(time.Second*100))
	if get, ok := cache.Get(key); get != data || !ok {
		t.Fatal("Could not retrieve a key that we should've")
	}

	cache.Del(key)
	if _, ok := cache.Get(key); ok {
		t.Fatal("Found key that was deleted")
	}
}

func TestDelTags(t *testing.T) {
	cache := newCache()
	data := "data"
	var key Key = "key"
	var tag Tag = "uno"
	tags := []Tag{tag, "dos"}

	cache.Set(key, tags, data, time.Now().Add(time.Second*100))
	cache.Del(key)

	stored := cache.tags[tag]
	if stored == nil {
		t.Fatal("Tag cleared to soon")
	}
	if _, ok := stored.t[key]; !ok {
		t.Fatal("Tag key cleared to soon")
	}

	// Clean tags
	cache.cleanTags(1e9)
	stored = cache.tags[tag]
	if stored == nil {
		t.Fatal("Tag cleared to soon")
	}
	if _, ok := stored.t[key]; !ok {
		t.Fatal("Tag key cleared to soon")
	}

	// Clean tag keys
	cache.cleanTagKeys(1e9)
	stored = cache.tags[tag]
	if stored == nil {
		t.Fatal("Tag cleared to soon")
	}
	if _, ok := stored.t[key]; ok {
		t.Fatal("Tag key not cleared")
	}

	// Clean tags
	cache.cleanTags(1e9)
	if stored == nil {
		t.Error("Tag not cleared")
	}
}

func TestDelByTag(t *testing.T) {
	cache := newCache()
	expires := time.Now().Add(time.Second * 100)

	cache.Set("uno", []Tag{"tag1", "tag2"}, "data", expires)
	cache.Set("dos", []Tag{"tag1"}, "data", expires)
	cache.Set("tres", []Tag{"tag2"}, "data", expires)

	for _, k := range []Key{"uno", "dos", "tres"} {
		if _, ok := cache.Get(k); !ok {
			t.Fatalf("Key %s should exist", k)
		}
	}

	cache.DelByTag("tag1")
	for _, k := range []Key{"uno", "dos"} {
		if _, ok := cache.Get(k); ok {
			t.Fatalf("Key %s should not exist", k)
		}
	}

	if _, ok := cache.Get("tres"); !ok {
		t.Fatalf("Key tres should exist")
	}
}

func TestDelByPrefix(t *testing.T) {
	cache := newCache()
	expires := time.Now().Add(time.Second * 100)

	cache.Set("prefix:lala", nil, "data", expires)
	cache.Set("prefix:lolo", nil, "data", expires)
	cache.Set("prefix:lili", nil, "data", expires)
	cache.Set("prefix:lulu", nil, "data", expires)
	cache.Set("some-key", nil, "data", expires)

	cache.DelByPrefix("prefix:")

	for i := range cache.data {
		if i != "some-key" {
			t.Fatalf("%s should not exist", i)
		}
	}

	if _, ok := cache.Get("some-key"); !ok {
		t.Fatalf("Key not should exist")
	}
}

func TestGetExpired(t *testing.T) {
	cache := newCache()
	expires := time.Now().Add(-time.Second * 100)
	cache.Set("uno", nil, "data", expires)
	if _, ok := cache.Get("uno"); ok {
		t.Fatal("Key should've expired")
	}
}

func TestGetTagKeys(t *testing.T) {
	cache := newCache()
	expires := time.Now().Add(-time.Second * 100)
	cache.Set("uno", []Tag{"tag1", "tag2"}, "data", expires)
	cache.Set("dos", []Tag{"tag1"}, "data", expires)
	cache.Set("tres", []Tag{"tag2"}, "data", expires)
	cache.Set("cuatro", []Tag{"tag3"}, "data", expires)
	enc := map[Key]int{
		"uno":  1,
		"tres": 1,
	}

	for _, k := range cache.GetTagKeys("tag2") {
		enc[k]--
	}

	for k, v := range enc {
		if v != 0 {
			t.Fatal(k)
		}
	}
}
