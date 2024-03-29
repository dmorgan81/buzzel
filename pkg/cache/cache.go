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
	"context"
	"errors"
	"io"
)

type Store string

func (s Store) String() string {
	return string(s)
}

const (
	AC  Store = "ac"
	CAS Store = "cas"
)

type Key string

func (k Key) String() string {
	return string(k)
}

type Cache interface {
	Exists(context.Context, Store, Key) error
	Reader(context.Context, Store, Key) (io.Reader, int64, error)
	Writer(context.Context, Store, Key) (io.Writer, error)
}

var ErrNotFound = errors.New("cache: not found")
