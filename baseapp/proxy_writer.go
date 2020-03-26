// Copyright 2020 Palantir Technologies, Inc.
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

package baseapp

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

// ProxyResponseWriter is a proxy around an http.ResponseWriter
// that counts bytes written to the wrapped ResponseWriter.
type ProxyResponseWriter interface {
	http.ResponseWriter

	// BytesWritten returns the total number of bytes sent to the client.
	Status() int

	// BytesWritten returns the total number of bytes sent to the client.
	BytesWritten() int64
}

func WrapWriter(w http.ResponseWriter) ProxyResponseWriter {
	_, cn := w.(http.CloseNotifier)
	_, fl := w.(http.Flusher)
	_, hj := w.(http.Hijacker)
	_, rf := w.(io.ReaderFrom)

	bp := basicProxy{ResponseWriter: w}
	if cn && fl && hj && rf {
		return &fancyProxy{bp}
	}
	if fl {
		return &flushProxy{bp}
	}
	return &bp
}

type basicProxy struct {
	http.ResponseWriter
	code         int
	bytesWritten int64
}

func (b *basicProxy) WriteHeader(code int) {
	if b.code == 0 {
		b.code = code
	}
	b.ResponseWriter.WriteHeader(code)
}

func (b *basicProxy) Write(buf []byte) (int, error) {
	if b.code == 0 {
		b.code = http.StatusOK
	}
	n, err := b.ResponseWriter.Write(buf)
	b.bytesWritten += int64(n)
	return n, err
}

func (b *basicProxy) Status() int {
	return b.code
}

func (b *basicProxy) BytesWritten() int64 {
	return b.bytesWritten
}

// fancyProxy is a writer that additionally satisfies http.CloseNotifier,
// http.Flusher, http.Hijacker, and io.ReaderFrom. It exists for the common case
// of wrapping the http.ResponseWriter that package http gives you, in order to
// make the proxied object support the full method set of the proxied object.
type fancyProxy struct {
	basicProxy
}

func (f *fancyProxy) CloseNotify() <-chan bool {
	cn := f.basicProxy.ResponseWriter.(http.CloseNotifier)
	return cn.CloseNotify()
}
func (f *fancyProxy) Flush() {
	fl := f.basicProxy.ResponseWriter.(http.Flusher)
	fl.Flush()
}
func (f *fancyProxy) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj := f.basicProxy.ResponseWriter.(http.Hijacker)
	return hj.Hijack()
}
func (f *fancyProxy) ReadFrom(r io.Reader) (int64, error) {
	if f.code == 0 {
		f.code = http.StatusOK
	}
	rf := f.basicProxy.ResponseWriter.(io.ReaderFrom)
	n, err := rf.ReadFrom(r)
	f.bytesWritten += n
	return n, err
}

var _ http.CloseNotifier = &fancyProxy{}
var _ http.Flusher = &fancyProxy{}
var _ http.Hijacker = &fancyProxy{}
var _ io.ReaderFrom = &fancyProxy{}

type flushProxy struct {
	basicProxy
}

func (f *flushProxy) Flush() {
	fl := f.basicProxy.ResponseWriter.(http.Flusher)
	fl.Flush()
}

var _ http.Flusher = &flushProxy{}
