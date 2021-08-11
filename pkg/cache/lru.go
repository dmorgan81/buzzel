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
	"container/list"
	"context"
	"io"
	"path"
	"sync"

	health "github.com/etherlabsio/healthcheck/v2"
	"github.com/rs/zerolog"
)

type LRU struct {
	cache Cache
	lock  sync.Mutex
	ll    *list.List
	mp    map[string]*list.Element
	size  int64
	max   int64
}

type entry struct {
	store Store
	key   Key
	data  []byte
}

var _ Cache = &LRU{}

func NewLRUCache(cache Cache, max int64) *LRU {
	return &LRU{cache: cache, ll: list.New(), mp: make(map[string]*list.Element), max: max}
}

func resolve(store Store, key Key) string {
	return path.Join(string(store), string(key))
}

func (c *LRU) touch(store Store, key Key) ([]byte, bool) {
	if el, ok := c.mp[resolve(store, key)]; ok {
		c.ll.MoveToFront(el)
		return el.Value.(*entry).data, true
	}
	return nil, false
}

func (c *LRU) evict(store Store, key Key) {
	path := resolve(store, key)
	if el, ok := c.mp[path]; ok {
		en := c.ll.Remove(el).(*entry)
		delete(c.mp, path)
		c.size -= int64(len(en.data))
	}
}

func (c *LRU) pop() (Store, Key, []byte) {
	en := c.ll.Remove(c.ll.Back()).(*entry)
	delete(c.mp, resolve(en.store, en.key))
	c.size -= int64(len(en.data))
	return en.store, en.key, en.data
}

func (c *LRU) push(store Store, key Key, data []byte) {
	c.mp[resolve(store, key)] = c.ll.PushFront(&entry{store, key, data})
	c.size += int64(len(data))
}

func (c *LRU) load(ctx context.Context, store Store, key Key) (io.Reader, int64, error) {
	log := zerolog.Ctx(ctx).With().
		Stringer("store", store).
		Stringer("key", key).
		Logger()
	log.Debug().Caller().Msg("cache miss")

	reader, size, err := c.cache.Reader(ctx, store, key)
	if err != nil {
		return nil, -1, err
	}

	if size > c.max {
		return reader, size, nil
	}

	for size+c.size > c.max {
		store, key, data := c.pop()
		log.Debug().Caller().
			Stringer("store", store).
			Stringer("key", key).
			Int64("size", int64(len(data))).
			Msg("cache evict")
	}

	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, reader); err != nil {
		return nil, -1, err
	}

	data := buf.Bytes()
	c.push(store, key, data)
	log.Debug().Caller().
		Int64("size", c.size).
		Int64("max", c.max).
		Msg("cache load")
	return bytes.NewBuffer(data), size, nil
}

func (c *LRU) Exists(ctx context.Context, store Store, key Key) error {
	log := zerolog.Ctx(ctx).With().
		Stringer("store", store).
		Stringer("key", key).
		Logger()
	c.lock.Lock()
	defer c.lock.Unlock()

	if _, ok := c.touch(store, key); ok {
		log.Debug().Caller().Msg("cache hit")
		return nil
	}

	if err := c.cache.Exists(ctx, store, key); err != nil {
		return err
	}

	_, _, err := c.load(ctx, store, key)
	return err
}

func (c *LRU) Reader(ctx context.Context, store Store, key Key) (io.Reader, int64, error) {
	log := zerolog.Ctx(ctx).With().
		Stringer("store", store).
		Stringer("key", key).
		Logger()
	c.lock.Lock()
	defer c.lock.Unlock()

	if data, ok := c.touch(store, key); ok {
		log.Debug().Caller().Msg("cache hit")
		return bytes.NewBuffer(data), int64(len(data)), nil
	}

	return c.load(ctx, store, key)
}

func (c *LRU) Writer(ctx context.Context, store Store, key Key) (io.Writer, error) {
	c.lock.Lock()
	c.evict(store, key)
	c.lock.Unlock()
	return c.cache.Writer(ctx, store, key)
}

var _ health.Checker = &LRU{}

func (c *LRU) Check(ctx context.Context) error {
	if checker, ok := c.cache.(health.Checker); ok {
		return checker.Check(ctx)
	}
	return nil
}
