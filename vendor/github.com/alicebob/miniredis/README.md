# Miniredis

Pure Go Redis test server, used in Go unittests.


##

Sometimes you want to test code which uses Redis, without making it a full-blown
integration test.
Miniredis implements (parts of) the Redis server, to be used in unittests. It
enables a simple, cheap, in-memory, Redis replacement, with a real TCP interface. Think of it as the Redis version of `net/http/httptest`.

It saves you from using mock code, and since the redis server lives in the
test process you can query for values directly, without going through the server
stack.

There are no dependencies on external binaries, so you can easily integrate it in automated build processes.

## Changelog

### 2.5.0

Added ZPopMin and ZPopMax

### v2.4.6

support for TIME (thanks @leon-barrett and @lirao)
support for ZREVRANGEBYLEX
fix for SINTER (thanks @robstein)
updates for latest redis

### 2.4.4

Fixed nil Lua return value (#43)

### 2.4.3

Fixed using Lua with authenticated redis.

### 2.4.2

Changed redigo import path.

### 2.4

Minor cleanups. Miniredis now requires Go >= 1.9 (only for the tests. If you don't run the tests you can use an older Go version).

### 2.3.1

Lua changes: added `cjson` library, and `redis.sha1hex()`.

### 2.3

Added the `EVAL`, `EVALSHA`, and `SCRIPT` commands. Uses a pure Go Lua interpreter. Please open an issue if there are problems with any Lua code.

### 2.2

Introduced `StartAddr()`.

### 2.1

Internal cleanups. No changes in functionality.

### 2.0

2.0.0 improves TTLs to be `time.Duration` values. `.Expire()` is removed and
replaced by `.TTL()`, which returns the TTL as a `time.Duration`.
This should be the change needed to upgrade:

1.0:

    m.Expire() == 4
   
2.0:

    m.TTL() == 4 * time.Second

Furthermore, `.SetTime()` is added to help with `EXPIREAT` commands, and `.FastForward()` is introduced to test keys expiration.


## Commands

Implemented commands:

 - Connection (complete)
   - AUTH -- see RequireAuth()
   - ECHO
   - PING
   - SELECT
   - QUIT
 - Key 
   - DEL
   - EXISTS
   - EXPIRE
   - EXPIREAT
   - KEYS
   - MOVE
   - PERSIST
   - PEXPIRE
   - PEXPIREAT
   - PTTL
   - RENAME
   - RENAMENX
   - RANDOMKEY -- call math.rand.Seed(...) once before using.
   - TTL
   - TYPE
   - SCAN
 - Transactions (complete)
   - DISCARD
   - EXEC
   - MULTI
   - UNWATCH
   - WATCH
 - Server
   - DBSIZE
   - FLUSHALL
   - FLUSHDB
   - TIME -- returns time.Now() or value set by SetTime()
 - String keys (complete)
   - APPEND
   - BITCOUNT
   - BITOP
   - BITPOS
   - DECR
   - DECRBY
   - GET
   - GETBIT
   - GETRANGE
   - GETSET
   - INCR
   - INCRBY
   - INCRBYFLOAT
   - MGET
   - MSET
   - MSETNX
   - PSETEX
   - SET
   - SETBIT
   - SETEX
   - SETNX
   - SETRANGE
   - STRLEN
 - Hash keys (complete)
   - HDEL
   - HEXISTS
   - HGET
   - HGETALL
   - HINCRBY
   - HINCRBYFLOAT
   - HKEYS
   - HLEN
   - HMGET
   - HMSET
   - HSET
   - HSETNX
   - HVALS
   - HSCAN
 - List keys (complete)
   - BLPOP
   - BRPOP
   - BRPOPLPUSH
   - LINDEX
   - LINSERT
   - LLEN
   - LPOP
   - LPUSH
   - LPUSHX
   - LRANGE
   - LREM
   - LSET
   - LTRIM
   - RPOP
   - RPOPLPUSH
   - RPUSH
   - RPUSHX
 - Set keys (complete)
   - SADD
   - SCARD
   - SDIFF
   - SDIFFSTORE
   - SINTER
   - SINTERSTORE
   - SISMEMBER
   - SMEMBERS
   - SMOVE
   - SPOP -- call math.rand.Seed(...) once before using.
   - SRANDMEMBER -- call math.rand.Seed(...) once before using.
   - SREM
   - SUNION
   - SUNIONSTORE
   - SSCAN
 - Sorted Set keys (complete)
   - ZADD
   - ZCARD
   - ZCOUNT
   - ZINCRBY
   - ZINTERSTORE
   - ZLEXCOUNT
   - ZPOPMIN
   - ZPOPMAX
   - ZRANGE
   - ZRANGEBYLEX
   - ZRANGEBYSCORE
   - ZRANK
   - ZREM
   - ZREMRANGEBYLEX
   - ZREMRANGEBYRANK
   - ZREMRANGEBYSCORE
   - ZREVRANGE
   - ZREVRANGEBYLEX
   - ZREVRANGEBYSCORE
   - ZREVRANK
   - ZSCORE
   - ZUNIONSTORE
   - ZSCAN
 - Scripting
   - EVAL
   - EVALSHA
   - SCRIPT LOAD
   - SCRIPT EXISTS
   - SCRIPT FLUSH

## TTLs, key expiration, and time

Since miniredis is intended to be used in unittests TTLs don't decrease
automatically. You can use `TTL()` to get the TTL (as a time.Duration) of a
key. It will return 0 when no TTL is set.

`m.FastForward(d)` can be used to decrement all TTLs. All TTLs which become <=
0 will be removed.

EXPIREAT and PEXPIREAT values will be
converted to a duration. For that you can either set m.SetTime(t) to use that
time as the base for the (P)EXPIREAT conversion, or don't call SetTime(), in
which case time.Now() will be used.

SetTime() also sets the value returned by TIME, which defaults to time.Now().
It is not updated by FastForward, only by SetTime.

## Example

``` Go
func TestSomething(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	// Optionally set some keys your code expects:
	s.Set("foo", "bar")
	s.HSet("some", "other", "key")

	// Run your code and see if it behaves.
	// An example using the redigo library from "github.com/gomodule/redigo/redis":
	c, err := redis.Dial("tcp", s.Addr())
	_, err = c.Do("SET", "foo", "bar")

	// Optionally check values in redis...
	if got, err := s.Get("foo"); err != nil || got != "bar" {
		t.Error("'foo' has the wrong value")
	}
	// ... or use a helper for that:
	s.CheckGet(t, "foo", "bar")

	// TTL and expiration:
	s.Set("foo", "bar")
	s.SetTTL("foo", 10*time.Second)
	s.FastForward(11 * time.Second)
	if s.Exists("foo") {
		t.Fatal("'foo' should not have existed anymore")
	}
}
```

## Not supported

Commands which will probably not be implemented:

 - CLUSTER (all)
    - ~~CLUSTER *~~
    - ~~READONLY~~
    - ~~READWRITE~~
 - GEO (all) -- unless someone needs these
    - ~~GEOADD~~
    - ~~GEODIST~~
    - ~~GEOHASH~~
    - ~~GEOPOS~~
    - ~~GEORADIUS~~
    - ~~GEORADIUSBYMEMBER~~
 - HyperLogLog (all) -- unless someone needs these
    - ~~PFADD~~
    - ~~PFCOUNT~~
    - ~~PFMERGE~~
 - Key
    - ~~DUMP~~
    - ~~MIGRATE~~
    - ~~OBJECT~~
    - ~~RESTORE~~
    - ~~WAIT~~
 - Pub/Sub (all)
    - ~~PSUBSCRIBE~~
    - ~~PUBLISH~~
    - ~~PUBSUB~~
    - ~~PUNSUBSCRIBE~~
    - ~~SUBSCRIBE~~
    - ~~UNSUBSCRIBE~~
 - Scripting
    - ~~SCRIPT DEBUG~~
    - ~~SCRIPT KILL~~
 - Server
    - ~~BGSAVE~~
    - ~~BGWRITEAOF~~
    - ~~CLIENT *~~
    - ~~COMMAND *~~
    - ~~CONFIG *~~
    - ~~DEBUG *~~
    - ~~INFO~~
    - ~~LASTSAVE~~
    - ~~MONITOR~~
    - ~~ROLE~~
    - ~~SAVE~~
    - ~~SHUTDOWN~~
    - ~~SLAVEOF~~
    - ~~SLOWLOG~~
    - ~~SYNC~~
    

## &c.

Tests are run against Redis 5.0.3. The [./integration](./integration/) subdir
compares miniredis against a real redis instance.


[![Build Status](https://travis-ci.org/alicebob/miniredis.svg?branch=master)](https://travis-ci.org/alicebob/miniredis) 
[![GoDoc](https://godoc.org/github.com/alicebob/miniredis?status.svg)](https://godoc.org/github.com/alicebob/miniredis)
