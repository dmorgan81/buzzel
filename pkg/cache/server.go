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
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/justinas/alice"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

func NewServer(addr string, cache Cache) *http.Server {
	chain := alice.New(hlog.NewHandler(log.Logger), gziphandler.GzipHandler)
	chain = chain.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Stringer("url", r.URL).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("")
	}))
	mux := http.NewServeMux()
	mux.Handle("/", chain.Then(&handler{Cache: cache}))
	return &http.Server{Addr: addr, Handler: mux}
}

type handler struct {
	Cache
}

var _ http.Handler = &handler{}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodHead:
		h.head(w, r)
	case http.MethodGet:
		h.get(w, r)
	case http.MethodPut:
		h.put(w, r)
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func parseRequestURL(url *url.URL) (Store, Key, error) {
	path := url.Path
	if strings.HasPrefix(path, "/ac/") || strings.HasPrefix(path, "/cas/") {
		path = path[1:]

		parts := strings.Split(path, "/")
		if len(parts) != 2 {
			return "", "", ErrNotFound
		}

		var store Store
		if parts[0] == string(AC) {
			store = AC
		} else if parts[0] == string(CAS) {
			store = CAS
		} else {
			return "", "", ErrNotFound
		}

		key := parts[1]
		key = key[:2] + "/" + key
		return store, Key(key), nil
	}
	return "", "", ErrNotFound
}

func handleHttpError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
	} else {
		hlog.FromRequest(r).Err(err).Stack().Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *handler) head(w http.ResponseWriter, r *http.Request) {
	store, key, err := parseRequestURL(r.URL)
	if err != nil {
		handleHttpError(w, r, err)
		return
	}

	if err := h.Exists(r.Context(), store, key); err != nil {
		handleHttpError(w, r, err)
	}
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	store, key, err := parseRequestURL(r.URL)
	if err != nil {
		handleHttpError(w, r, err)
		return
	}

	if err := h.Exists(r.Context(), store, key); err != nil {
		handleHttpError(w, r, err)
		return
	}

	reader, size, err := h.Reader(r.Context(), store, key)
	if err != nil {
		handleHttpError(w, r, err)
		return
	}
	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}

	w.Header().Add("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Add("Content-Type", "application/octect-stream")
	if size == 0 {
		w.WriteHeader(http.StatusOK)
	}
	io.Copy(w, reader)
}

func (h *handler) put(w http.ResponseWriter, r *http.Request) {
	store, key, err := parseRequestURL(r.URL)
	if err != nil {
		handleHttpError(w, r, err)
		return
	}

	writer, err := h.Writer(r.Context(), store, key)
	if err != nil && !errors.Is(err, ErrNotFound) {
		handleHttpError(w, r, err)
		return
	}
	if closer, ok := writer.(io.Closer); ok {
		defer closer.Close()
	}

	if written, err := io.Copy(writer, r.Body); err != nil {
		handleHttpError(w, r, err)
	} else {
		hlog.FromRequest(r).Debug().Caller().
			Stringer("store", store).
			Stringer("key", key).
			Int64("size", written).
			Send()
	}
	w.WriteHeader(http.StatusOK)
}
