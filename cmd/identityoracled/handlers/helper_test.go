package handlers

import (
	"bytes"
	"net/http"
)

type MockResponseWriter struct {
	header     http.Header
	StatusCode int
	buffer     bytes.Buffer
}

func (rw *MockResponseWriter) Header() http.Header {
	return rw.header
}

func (rw *MockResponseWriter) Write(b []byte) (int, error) {
	return rw.buffer.Write(b)
}

func (rw *MockResponseWriter) WriteHeader(statusCode int) {
	rw.StatusCode = statusCode
}
