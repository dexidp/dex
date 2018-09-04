package bigcache

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

var message = blob('a', 256)

func BenchmarkWriteToCacheWith1Shard(b *testing.B) {
	writeToCache(b, 1, 100*time.Second, b.N)
}

func BenchmarkWriteToLimitedCacheWithSmallInitSizeAnd1Shard(b *testing.B) {
	m := blob('a', 1024)
	cache, _ := NewBigCache(Config{
		Shards:             1,
		LifeWindow:         100 * time.Second,
		MaxEntriesInWindow: 100,
		MaxEntrySize:       256,
		HardMaxCacheSize:   1,
	})

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.Set(fmt.Sprintf("key-%d", i), m)
	}
}

func BenchmarkWriteToUnlimitedCacheWithSmallInitSizeAnd1Shard(b *testing.B) {
	m := blob('a', 1024)
	cache, _ := NewBigCache(Config{
		Shards:             1,
		LifeWindow:         100 * time.Second,
		MaxEntriesInWindow: 100,
		MaxEntrySize:       256,
	})

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.Set(fmt.Sprintf("key-%d", i), m)
	}
}

func BenchmarkWriteToCache(b *testing.B) {
	for _, shards := range []int{1, 512, 1024, 8192} {
		b.Run(fmt.Sprintf("%d-shards", shards), func(b *testing.B) {
			writeToCache(b, shards, 100*time.Second, b.N)
		})
	}
}

func BenchmarkReadFromCache(b *testing.B) {
	for _, shards := range []int{1, 512, 1024, 8192} {
		b.Run(fmt.Sprintf("%d-shards", shards), func(b *testing.B) {
			readFromCache(b, 1024)
		})
	}
}

func BenchmarkIterateOverCache(b *testing.B) {

	m := blob('a', 1)

	for _, shards := range []int{512, 1024, 8192} {
		b.Run(fmt.Sprintf("%d-shards", shards), func(b *testing.B) {
			cache, _ := NewBigCache(Config{
				Shards:             shards,
				LifeWindow:         1000 * time.Second,
				MaxEntriesInWindow: max(b.N, 100),
				MaxEntrySize:       500,
			})

			for i := 0; i < b.N; i++ {
				cache.Set(fmt.Sprintf("key-%d", i), m)
			}

			b.ResetTimer()
			it := cache.Iterator()

			b.RunParallel(func(pb *testing.PB) {
				b.ReportAllocs()

				for pb.Next() {
					if it.SetNext() {
						it.Value()
					}
				}
			})
		})
	}
}

func BenchmarkWriteToCacheWith1024ShardsAndSmallShardInitSize(b *testing.B) {
	writeToCache(b, 1024, 100*time.Second, 100)
}

func writeToCache(b *testing.B, shards int, lifeWindow time.Duration, requestsInLifeWindow int) {
	cache, _ := NewBigCache(Config{
		Shards:             shards,
		LifeWindow:         lifeWindow,
		MaxEntriesInWindow: max(requestsInLifeWindow, 100),
		MaxEntrySize:       500,
	})
	rand.Seed(time.Now().Unix())

	b.RunParallel(func(pb *testing.PB) {
		id := rand.Int()
		counter := 0

		b.ReportAllocs()
		for pb.Next() {
			cache.Set(fmt.Sprintf("key-%d-%d", id, counter), message)
			counter = counter + 1
		}
	})
}

func readFromCache(b *testing.B, shards int) {
	cache, _ := NewBigCache(Config{
		Shards:             shards,
		LifeWindow:         1000 * time.Second,
		MaxEntriesInWindow: max(b.N, 100),
		MaxEntrySize:       500,
	})
	for i := 0; i < b.N; i++ {
		cache.Set(strconv.Itoa(i), message)
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()

		for pb.Next() {
			cache.Get(strconv.Itoa(rand.Intn(b.N)))
		}
	})
}
