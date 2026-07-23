package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBodyLimitMultipartGetsBiggerBudget(t *testing.T) {
	var got int
	h := BodyLimit(1<<10, 1<<20)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = len(b)
	}))

	body := bytes.Repeat([]byte("x"), 4<<10)

	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=abc")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if got != len(body) {
		t.Fatalf("multipart terpotong: %d dari %d byte", got, len(body))
	}

	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if got == len(body) {
		t.Fatalf("json body tidak dibatasi: %d byte lolos", got)
	}
}
