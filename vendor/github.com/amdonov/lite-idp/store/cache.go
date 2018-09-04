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
	"time"

	"github.com/allegro/bigcache"
)

type Cache interface {
	Set(key string, entry []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

// Default to a big cache implementation
func New(duration time.Duration) (Cache, error) {
	cache, err := bigcache.NewBigCache(bigcache.DefaultConfig(duration))
	if err != nil {
		return nil, err
	}
	return &bigcacheStore{cache}, nil
}
