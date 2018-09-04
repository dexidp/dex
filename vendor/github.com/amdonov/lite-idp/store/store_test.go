// Copyright © 2017 Aaron Donovan <amdonov@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package store

import (
	"testing"
	"time"
)

func TestDelete(t *testing.T) {
	c := "content2"
	cache, err := New(5 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	cache.Set("test", []byte(c))
	data, err := cache.Get("test")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != c {
		t.Fatal("data did not match expected value")
	}
	err = cache.Delete("test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = cache.Get("test")
	if err == nil {
		t.Fatal("should not have returned value")
	}
}
