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
	"path"
	"regexp"
	"strconv"
	"time"

	"github.com/NYTimes/gziphandler"
	health "github.com/etherlabsio/healthcheck/v2"
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
	mux.Handle("/ac/", chain.Then(&handler{Cache: cache, store: AC}))
	mux.Handle("/cas/", chain.Then(&handler{Cache: cache, store: CAS}))

	if checker, ok := cache.(health.Checker); ok {
		mux.Handle("/healthz", health.Handler(health.WithChecker("cache", checker)))
	} else {
		mux.Handle("/healthz", health.Handler())
	}

	return &http.Server{Addr: addr, Handler: mux}
}

type handler struct {
	Cache
	store Store
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

var sha256 = regexp.MustCompile("^[a-f0-9]{64}$")

func keyFromRequest(r *http.Request) (Key, error) {
	p := path.Base(r.URL.Path)
	if !sha256.Match([]byte(p)) {
		return "", ErrNotFound
	}
	p = p[:2] + "/" + p
	return Key(p), nil
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
	key, err := keyFromRequest(r)
	if err != nil {
		handleHttpError(w, r, err)
	}

	if err := h.Exists(r.Context(), h.store, key); err != nil {
		handleHttpError(w, r, err)
	}
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	key, err := keyFromRequest(r)
	if err != nil {
		handleHttpError(w, r, err)
	}

	if err := h.Exists(r.Context(), h.store, key); err != nil {
		handleHttpError(w, r, err)
		return
	}

	reader, size, err := h.Reader(r.Context(), h.store, key)
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
	key, err := keyFromRequest(r)
	if err != nil {
		handleHttpError(w, r, err)
	}

	writer, err := h.Writer(r.Context(), h.store, key)
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
			Stringer("store", h.store).
			Stringer("key", key).
			Int64("size", written).
			Send()
	}
	w.WriteHeader(http.StatusOK)
}
