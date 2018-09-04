package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/allegro/bigcache"
	"github.com/coocood/freecache"
)

func gcPause() time.Duration {
	runtime.GC()
	var stats debug.GCStats
	debug.ReadGCStats(&stats)
	return stats.PauseTotal
}

const (
	entries   = 20000000
	valueSize = 100
)

func main() {
	debug.SetGCPercent(10)
	fmt.Println("Number of entries: ", entries)

	config := bigcache.Config{
		Shards:             256,
		LifeWindow:         100 * time.Minute,
		MaxEntriesInWindow: entries,
		MaxEntrySize:       200,
		Verbose:            true,
	}

	bigcache, _ := bigcache.NewBigCache(config)
	for i := 0; i < entries; i++ {
		key, val := generateKeyValue(i, valueSize)
		bigcache.Set(key, val)
	}

	firstKey, _ := generateKeyValue(1, valueSize)
	checkFirstElement(bigcache.Get(firstKey))

	fmt.Println("GC pause for bigcache: ", gcPause())
	bigcache = nil
	gcPause()

	//------------------------------------------

	freeCache := freecache.NewCache(entries * 200) //allocate entries * 200 bytes
	for i := 0; i < entries; i++ {
		key, val := generateKeyValue(i, valueSize)
		if err := freeCache.Set([]byte(key), val, 0); err != nil {
			fmt.Println("Error in set: ", err.Error())
		}
	}

	firstKey, _ = generateKeyValue(1, valueSize)
	checkFirstElement(freeCache.Get([]byte(firstKey)))

	if freeCache.OverwriteCount() != 0 {
		fmt.Println("Overwritten: ", freeCache.OverwriteCount())
	}
	fmt.Println("GC pause for freecache: ", gcPause())
	freeCache = nil
	gcPause()

	//------------------------------------------

	mapCache := make(map[string][]byte)
	for i := 0; i < entries; i++ {
		key, val := generateKeyValue(i, valueSize)
		mapCache[key] = val
	}
	fmt.Println("GC pause for map: ", gcPause())

}

func checkFirstElement(val []byte, err error) {
	_, expectedVal := generateKeyValue(1, valueSize)
	if err != nil {
		fmt.Println("Error in get: ", err.Error())
	} else if string(val) != string(expectedVal) {
		fmt.Println("Wrong first element: ", string(val))
	}
}

func generateKeyValue(index int, valSize int) (string, []byte) {
	key := fmt.Sprintf("key-%010d", index)
	fixedNumber := []byte(fmt.Sprintf("%010d", index))
	val := append(make([]byte, valSize-10), fixedNumber...)

	return key, val
}
