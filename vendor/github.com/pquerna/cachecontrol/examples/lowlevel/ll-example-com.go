package main

import (
	"github.com/pquerna/cachecontrol/cacheobject"

	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func main() {
	req, _ := http.NewRequest("GET", "http://www.example.com/", nil)

	res, _ := http.DefaultClient.Do(req)
	_, _ = ioutil.ReadAll(res.Body)

	reqDir, _ := cacheobject.ParseRequestCacheControl(req.Header.Get("Cache-Control"))

	resDir, _ := cacheobject.ParseResponseCacheControl(res.Header.Get("Cache-Control"))
	expiresHeader, _ := http.ParseTime(res.Header.Get("Expires"))
	dateHeader, _ := http.ParseTime(res.Header.Get("Date"))
	lastModifiedHeader, _ := http.ParseTime(res.Header.Get("Last-Modified"))

	obj := cacheobject.Object{
		RespDirectives:         resDir,
		RespHeaders:            res.Header,
		RespStatusCode:         res.StatusCode,
		RespExpiresHeader:      expiresHeader,
		RespDateHeader:         dateHeader,
		RespLastModifiedHeader: lastModifiedHeader,

		ReqDirectives: reqDir,
		ReqHeaders:    req.Header,
		ReqMethod:     req.Method,

		NowUTC: time.Now().UTC(),
	}
	rv := cacheobject.ObjectResults{}

	cacheobject.CachableObject(&obj, &rv)
	cacheobject.ExpirationObject(&obj, &rv)

	fmt.Println("Errors: ", rv.OutErr)
	fmt.Println("Reasons to not cache: ", rv.OutReasons)
	fmt.Println("Warning headers to add: ", rv.OutWarnings)
	fmt.Println("Expiration: ", rv.OutExpirationTime.String())
}
