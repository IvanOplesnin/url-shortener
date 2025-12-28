package handlers

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()

	var r io.Reader = resp.Body
	defer resp.Body.Close()

	if strings.Contains(resp.Header.Get(contentEncodingKey), "gzip") {
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			t.Fatalf("gzip.NewReader error: %v", err)
		}
		defer gr.Close()
		r = gr
	}

	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}
	return string(b)
}

func TestCompressGzip(t *testing.T) {
	type want struct {
		statusCode           int
		checkBody            func(*testing.T, string)
		checkContentEncoding func(*testing.T, string)
		checkVary            func(*testing.T, http.Header)
	}

	tests := []struct {
		name    string
		hand    http.HandlerFunc
		method  string
		gzipReq bool
		want    want
	}{
		{
			name: "gzip enabled json 201",
			hand: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(contentTypeKey, applicationJSONValue)
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"ok":true}`))
			}),
			method:  http.MethodGet,
			gzipReq: true,
			want: want{
				statusCode: http.StatusCreated,
				checkBody: func(t *testing.T, body string) {
					if body != `{"ok":true}` {
						t.Fatalf("body = %q, want %q", body, `{"ok":true}`)
					}
				},
				checkContentEncoding: func(t *testing.T, s string) {
					if !strings.Contains(s, "gzip") {
						t.Fatalf("Content-Encoding = %q, want gzip", s)
					}
				},
				checkVary: func(t *testing.T, h http.Header) {
					if !strings.Contains(h.Get("Vary"), acceptEncodingKey) {
						t.Fatalf("Vary = %q, want contains %q", h.Get("Vary"), acceptEncodingKey)
					}
				},
			},
		},

		{
			name: "no gzip when client does not accept",
			hand: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(contentTypeKey, textPlainValue)
				_, _ = w.Write([]byte("hello"))
			}),
			method:  http.MethodGet,
			gzipReq: false,
			want: want{
				statusCode: http.StatusOK,
				checkBody: func(t *testing.T, body string) {
					if body != "hello" {
						t.Fatalf("body = %q, want %q", body, "hello")
					}
				},
				checkContentEncoding: func(t *testing.T, s string) {
					if strings.Contains(s, "gzip") {
						t.Fatalf("Content-Encoding = %q, want empty (no gzip)", s)
					}
				},
			},
		},

		{
			name: "no gzip when content-type not supported",
			hand: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(contentTypeKey, "application/octet-stream")
				_, _ = w.Write([]byte("BIN"))
			}),
			method:  http.MethodGet,
			gzipReq: true,
			want: want{
				statusCode: http.StatusOK,
				checkBody: func(t *testing.T, body string) {
					if body != "BIN" {
						t.Fatalf("body = %q, want %q", body, "BIN")
					}
				},
				checkContentEncoding: func(t *testing.T, s string) {
					if strings.Contains(s, "gzip") {
						t.Fatalf("Content-Encoding = %q, want empty (no gzip)", s)
					}
				},
			},
		},

		{
			name: "no gzip for HEAD",
			hand: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(contentTypeKey, textPlainValue)
				_, _ = w.Write([]byte("hello"))
			}),
			method:  http.MethodHead,
			gzipReq: true,
			want: want{
				statusCode: http.StatusOK,
				checkContentEncoding: func(t *testing.T, s string) {
					if strings.Contains(s, "gzip") {
						t.Fatalf("Content-Encoding = %q, want empty (no gzip for HEAD)", s)
					}
				},
			},
		},

		{
			name: "no gzip for 204 no content",
			hand: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(contentTypeKey, textPlainValue)
				w.WriteHeader(http.StatusNoContent)
			}),
			method:  http.MethodGet,
			gzipReq: true,
			want: want{
				statusCode: http.StatusNoContent,
				checkContentEncoding: func(t *testing.T, s string) {
					if strings.Contains(s, "gzip") {
						t.Fatalf("Content-Encoding = %q, want empty (no gzip for 204)", s)
					}
				},
				checkBody: func(t *testing.T, body string) {
					if body != "" {
						t.Fatalf("body = %q, want empty", body)
					}
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(CompressGzip(tt.hand))
			defer ts.Close()

			req, _ := http.NewRequest(tt.method, ts.URL, nil)
			if tt.gzipReq {
				req.Header.Set(acceptEncodingKey, "gzip")
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Do error: %v", err)
			}
			if resp.StatusCode != tt.want.statusCode {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.want.statusCode)
			}
			if tt.want.checkContentEncoding != nil {
				tt.want.checkContentEncoding(t, resp.Header.Get(contentEncodingKey))
			}
			if tt.want.checkVary != nil {
				tt.want.checkVary(t, resp.Header)
			}

			if tt.method == http.MethodHead {
				_ = resp.Body.Close()
				return
			}

			body := readBody(t, resp)
			if tt.want.checkBody != nil {
				tt.want.checkBody(t, body)
			}
		})
	}

}
