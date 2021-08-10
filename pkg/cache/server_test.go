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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	assert := assert.New(t)
	specs := []struct {
		method string
		code   int
		sha    string
		req    []byte
		resp   []byte
	}{
		{http.MethodConnect, http.StatusMethodNotAllowed, "", nil, nil},
		{http.MethodGet, http.StatusNotFound, "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", nil, nil},
		{http.MethodPut, http.StatusOK, "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", []byte("foo"), nil},
		{http.MethodGet, http.StatusOK, "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", nil, []byte("foo")},
		{http.MethodHead, http.StatusOK, "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", nil, nil},
		{http.MethodPut, http.StatusOK, "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", []byte("bar"), nil},
		{http.MethodGet, http.StatusOK, "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", nil, []byte("bar")},
	}

	h := &handler{Cache: NewMemCache(), store: AC}
	for _, s := range specs {
		req := httptest.NewRequest(s.method, "/ac/"+s.sha, bytes.NewReader(s.req))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		resp := w.Result()
		body, _ := ioutil.ReadAll(resp.Body)

		assert.Equal(s.code, resp.StatusCode)
		if s.resp != nil {
			assert.Equal(s.resp, body)
		}
	}
}
