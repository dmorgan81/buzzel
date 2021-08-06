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
package disk

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/dmorgan81/buzzel/pkg/cache"
	"github.com/rs/zerolog"
)

var ErrKeyIsDir = errors.New("disk cache: key is dir")

type Cache string

var _ cache.Cache = Cache("")

func (c Cache) resolve(store cache.Store, key cache.Key) string {
	return filepath.Join(string(c), filepath.FromSlash(filepath.Join(string(store), string(key))))
}

func (c Cache) Exists(ctx context.Context, store cache.Store, key cache.Key) error {
	path := c.resolve(store, key)
	log := zerolog.Ctx(ctx).With().Caller().Logger()
	log.Debug().Str("path", path).Send()

	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return cache.ErrNotFound
	} else if err != nil {
		return err
	}

	if info.IsDir() {
		return ErrKeyIsDir
	}
	return nil
}

func (c Cache) Reader(ctx context.Context, store cache.Store, key cache.Key) (io.Reader, int64, error) {
	path := c.resolve(store, key)
	log := zerolog.Ctx(ctx).With().Caller().Logger()
	log.Debug().Str("path", path).Send()

	file, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		return nil, -1, err
	}

	info, err := file.Stat()
	if err != nil {
		return nil, -1, err
	}

	return file, info.Size(), nil
}

func (c Cache) Writer(ctx context.Context, store cache.Store, key cache.Key) (io.Writer, error) {
	path := c.resolve(store, key)
	log := zerolog.Ctx(ctx).With().Caller().Logger()
	log.Debug().Str("path", path).Send()

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}

	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
}
