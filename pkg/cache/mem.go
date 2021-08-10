/*
Copyright © 2021 David Morgan <dmorgan81@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cache

import (
	"bytes"
	"context"
	"io"
	"sync"
)

type MemCache struct {
	mp   map[string][]byte
	lock sync.RWMutex
}

var _ Cache = &MemCache{}

func NewMemCache() *MemCache {
	return &MemCache{mp: make(map[string][]byte)}
}

func (c *MemCache) Exists(_ context.Context, store Store, key Key) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if _, ok := c.mp[resolve(store, key)]; ok {
		return nil
	}
	return ErrNotFound
}

func (c *MemCache) Reader(_ context.Context, store Store, key Key) (io.Reader, int64, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if data, ok := c.mp[resolve(store, key)]; ok {
		return bytes.NewBuffer(data), int64(len(data)), nil
	}
	return nil, -1, ErrNotFound
}

type memwriter struct {
	buf   *bytes.Buffer
	cache *MemCache
	store Store
	key   Key
}

func (w *memwriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *memwriter) Close() error {
	w.cache.lock.Lock()
	defer w.cache.lock.Unlock()

	data := w.buf.Bytes()
	w.cache.mp[resolve(w.store, w.key)] = data
	return nil
}

func (c *MemCache) Writer(_ context.Context, store Store, key Key) (io.Writer, error) {
	return &memwriter{buf: &bytes.Buffer{}, cache: c, store: store, key: key}, nil
}
