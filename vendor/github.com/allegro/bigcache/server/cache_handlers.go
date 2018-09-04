package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func cacheIndexHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getCacheHandler(w, r)
		case http.MethodPut:
			putCacheHandler(w, r)
		case http.MethodDelete:
			deleteCacheHandler(w, r)
		}
	})
}

// handles get requests.
func getCacheHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Path[len(cachePath):]
	if target == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("can't get a key if there is no key."))
		log.Print("empty request.")
		return
	}
	entry, err := cache.Get(target)
	if err != nil {
		errMsg := (err).Error()
		if strings.Contains(errMsg, "not found") {
			log.Print(err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(entry)
}

func putCacheHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Path[len(cachePath):]
	if target == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("can't put a key if there is no key."))
		log.Print("empty request.")
		return
	}

	entry, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := cache.Set(target, []byte(entry)); err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("stored \"%s\" in cache.", target)
	w.WriteHeader(http.StatusCreated)
}

// delete cache objects.
func deleteCacheHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Path[len(cachePath):]
	if err := cache.Delete(target); err != nil {
		if strings.Contains((err).Error(), "not found") {
			w.WriteHeader(http.StatusNotFound)
			log.Printf("%s not found.", target)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("internal cache error: %s", err)
	}
	// this is what the RFC says to use when calling DELETE.
	w.WriteHeader(http.StatusOK)
	return
}
