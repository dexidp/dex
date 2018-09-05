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
	"github.com/allegro/bigcache"
)

type bigcacheStore struct {
	cache *bigcache.BigCache
}

func (b *bigcacheStore) Set(key string, entry []byte) error {
	return b.cache.Set(key, entry)
}

func (b *bigcacheStore) Get(key string) ([]byte, error) {
	entry, err := b.cache.Get(key)
	if len(entry) == 7 {
		// this might be a deleted key
		if "DELETED" == string(entry) {
			return nil, &bigcache.EntryNotFoundError{}
		}
	}
	return entry, err
}

func (b *bigcacheStore) Delete(key string) error {
	return b.Set(key, []byte("DELETED"))
}
