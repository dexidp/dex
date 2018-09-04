package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/allegro/bigcache"
)

const (
	testBaseString = "http://bigcache.org"
)

func testCacheSetup() {
	cache, _ = bigcache.NewBigCache(bigcache.Config{
		Shards:             1024,
		LifeWindow:         10 * time.Minute,
		MaxEntriesInWindow: 1000 * 10 * 60,
		MaxEntrySize:       500,
		Verbose:            true,
		HardMaxCacheSize:   8192,
		OnRemove:           nil,
	})
}

func TestMain(m *testing.M) {
	testCacheSetup()
	m.Run()
}

func TestGetWithNoKey(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", testBaseString+"/api/v1/cache/", nil)
	rr := httptest.NewRecorder()

	getCacheHandler(rr, req)
	resp := rr.Result()

	if resp.StatusCode != 400 {
		t.Errorf("want: 400; got: %d", resp.StatusCode)
	}
}

func TestGetWithMissingKey(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", testBaseString+"/api/v1/cache/doesNotExist", nil)
	rr := httptest.NewRecorder()

	getCacheHandler(rr, req)
	resp := rr.Result()

	if resp.StatusCode != 404 {
		t.Errorf("want: 404; got: %d", resp.StatusCode)
	}
}

func TestGetKey(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", testBaseString+"/api/v1/cache/getKey", nil)
	rr := httptest.NewRecorder()

	// set something.
	cache.Set("getKey", []byte("123"))

	getCacheHandler(rr, req)
	resp := rr.Result()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("cannot deserialise test response: %s", err)
	}

	if string(body) != "123" {
		t.Errorf("want: 123; got: %s.\n\tcan't get existing key getKey.", string(body))
	}
}

func TestPutKey(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("PUT", testBaseString+"/api/v1/cache/putKey", bytes.NewBuffer([]byte("123")))
	rr := httptest.NewRecorder()

	putCacheHandler(rr, req)

	testPutKeyResult, err := cache.Get("putKey")
	if err != nil {
		t.Errorf("error returning cache entry: %s", err)
	}

	if string(testPutKeyResult) != "123" {
		t.Errorf("want: 123; got: %s.\n\tcan't get PUT key putKey.", string(testPutKeyResult))
	}
}

func TestPutEmptyKey(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("PUT", testBaseString+"/api/v1/cache/", bytes.NewBuffer([]byte("123")))
	rr := httptest.NewRecorder()

	putCacheHandler(rr, req)
	resp := rr.Result()

	if resp.StatusCode != 400 {
		t.Errorf("want: 400; got: %d.\n\tempty key insertion should return with 400", resp.StatusCode)
	}
}

func TestDeleteEmptyKey(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("DELETE", testBaseString+"/api/v1/cache/", bytes.NewBuffer([]byte("123")))
	rr := httptest.NewRecorder()

	deleteCacheHandler(rr, req)
	resp := rr.Result()

	if resp.StatusCode != 404 {
		t.Errorf("want: 404; got: %d.\n\tapparently we're trying to delete empty keys.", resp.StatusCode)
	}
}

func TestDeleteInvalidKey(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("DELETE", testBaseString+"/api/v1/cache/invalidDeleteKey", bytes.NewBuffer([]byte("123")))
	rr := httptest.NewRecorder()

	deleteCacheHandler(rr, req)
	resp := rr.Result()

	if resp.StatusCode != 404 {
		t.Errorf("want: 404; got: %d.\n\tapparently we're trying to delete invalid keys.", resp.StatusCode)
	}
}

func TestDeleteKey(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("DELETE", testBaseString+"/api/v1/cache/testDeleteKey", bytes.NewBuffer([]byte("123")))
	rr := httptest.NewRecorder()

	if err := cache.Set("testDeleteKey", []byte("123")); err != nil {
		t.Errorf("can't set key for testing. %s", err)
	}

	deleteCacheHandler(rr, req)
	resp := rr.Result()

	if resp.StatusCode != 200 {
		t.Errorf("want: 200; got: %d.\n\tcan't delete keys.", resp.StatusCode)
	}
}

func TestGetStats(t *testing.T) {
	t.Parallel()
	var testStats bigcache.Stats

	req := httptest.NewRequest("GET", testBaseString+"/api/v1/stats", nil)
	rr := httptest.NewRecorder()

	// manually enter a key so there are some stats. get it so there's at least 1 hit.
	if err := cache.Set("incrementStats", []byte("123")); err != nil {
		t.Errorf("error setting cache value. error %s", err)
	}
	// it's okay if this fails, since we'll catch it downstream.
	if _, err := cache.Get("incrementStats"); err != nil {
		t.Errorf("can't find incrementStats. error: %s", err)
	}

	getCacheStatsHandler(rr, req)
	resp := rr.Result()

	if err := json.NewDecoder(resp.Body).Decode(&testStats); err != nil {
		t.Errorf("error decoding cache stats. error: %s", err)
	}

	if testStats.Hits == 0 {
		t.Errorf("want: > 0; got: 0.\n\thandler not properly returning stats info.")
	}
}
