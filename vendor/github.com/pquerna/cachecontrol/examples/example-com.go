package main

import (
	"github.com/pquerna/cachecontrol"

	"fmt"
	"io/ioutil"
	"net/http"
)

func main() {
	req, _ := http.NewRequest("GET", "http://www.example.com/", nil)

	res, _ := http.DefaultClient.Do(req)
	_, _ = ioutil.ReadAll(res.Body)

	reasons, expires, _ := cachecontrol.CachableResponse(req, res, cachecontrol.Options{})

	fmt.Println("Reasons to not cache: ", reasons)
	fmt.Println("Expiration: ", expires.String())
}
