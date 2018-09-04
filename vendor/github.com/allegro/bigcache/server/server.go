package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/allegro/bigcache"
)

const (
	// base HTTP paths.
	apiVersion  = "v1"
	apiBasePath = "/api/" + apiVersion + "/"

	// path to cache.
	cachePath = apiBasePath + "cache/"
	statsPath = apiBasePath + "stats"

	// server version.
	version = "1.0.0"
)

var (
	port    int
	logfile string
	ver     bool

	// cache-specific settings.
	cache  *bigcache.BigCache
	config = bigcache.Config{}
)

func init() {
	flag.BoolVar(&config.Verbose, "v", false, "Verbose logging.")
	flag.IntVar(&config.Shards, "shards", 1024, "Number of shards for the cache.")
	flag.IntVar(&config.MaxEntriesInWindow, "maxInWindow", 1000*10*60, "Used only in initial memory allocation.")
	flag.DurationVar(&config.LifeWindow, "lifetime", 100000*100000*60, "Lifetime of each cache object.")
	flag.IntVar(&config.HardMaxCacheSize, "max", 8192, "Maximum amount of data in the cache in MB.")
	flag.IntVar(&config.MaxEntrySize, "maxShardEntrySize", 500, "The maximum size of each object stored in a shard. Used only in initial memory allocation.")
	flag.IntVar(&port, "port", 9090, "The port to listen on.")
	flag.StringVar(&logfile, "logfile", "", "Location of the logfile.")
	flag.BoolVar(&ver, "version", false, "Print server version.")
}

func main() {
	flag.Parse()

	if ver {
		fmt.Printf("BigCache HTTP Server v%s", version)
		os.Exit(0)
	}

	var logger *log.Logger

	if logfile == "" {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	} else {
		f, err := os.OpenFile(logfile, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
		logger = log.New(f, "", log.LstdFlags)
	}

	var err error
	cache, err = bigcache.NewBigCache(config)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Print("cache initialised.")

	// let the middleware log.
	http.Handle(cachePath, serviceLoader(cacheIndexHandler(), requestMetrics(logger)))
	http.Handle(statsPath, serviceLoader(statsIndexHandler(), requestMetrics(logger)))

	logger.Printf("starting server on :%d", port)

	strPort := ":" + strconv.Itoa(port)
	log.Fatal("ListenAndServe: ", http.ListenAndServe(strPort, nil))
}
