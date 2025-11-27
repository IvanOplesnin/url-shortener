package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	st "github.com/IvanOplesnin/url-shortener/internal/repository"
)

type fakeStorage struct {
	searchResult st.ShortURL
	searchErr    error

	addErr error

	searchCalls int
	addCalls    int

	lastSearchURL st.URL
	lastAddID     st.ShortURL
	lastAddURL    st.URL
}

func (f *fakeStorage) Search(u st.URL) (st.ShortURL, error) {
	f.searchCalls++
	f.lastSearchURL = u
	return f.searchResult, f.searchErr
}

func (f *fakeStorage) Add(id st.ShortURL, u st.URL) error {
	f.addCalls++
	f.lastAddID = id
	f.lastAddURL = u
	return f.addErr
}

func (f *fakeStorage) Get(id st.ShortURL) (st.URL, error) {
	return "", errors.New("not implemented")
}
// TestShortenLinkHandler проверяет обработчик сокращения ссылок.
//
// Каждый тестовый случай проверяет:
//   - Правильность кода ответа HTTP
//   - Корректность заголовка Content-Type
//   - Проверку тела ответа через функцию bodyCheck
//   - Количество вызовов методов хранилища Search и Add
//
// Поддерживаемые сценарии:
//   - Добавление новой ссылки в пустое хранилище
//   - Поиск уже сокращённой ссылки
//   - Обработка неподдерживаемого Content-Type
func TestShortenLinkHandler(t *testing.T) {
	baseURL := "http://localhost:8080"

	type want struct {
		statusCode  int
		contentType string
		bodyCheck   func(t *testing.T, body string)
		searchCalls int
		addCalls    int
	}

	tests := []struct {
		name        string
		method      string
		body        string
		contentType string
		storage     *fakeStorage
		want        want
	}{
		{
			name:        "new link in empty storage",
			method:      http.MethodPost,
			body:        "https://google.com",
			contentType: "text/plain",
			storage: &fakeStorage{
				searchErr: st.ErrNotFoundURL,
				addErr:    nil,
			},
			want: want{
				statusCode:  http.StatusCreated,
				contentType: "text/plain",
				bodyCheck: func(t *testing.T, body string) {
					if !strings.HasPrefix(body, baseURL+"/") {
						t.Fatalf("expected body to start with %q, got %q", baseURL+"/", body)
					}
				},
				searchCalls: 1,
				addCalls:    1,
			},
		},
		{
			name:        "existing link in storage",
			method:      http.MethodPost,
			body:        "https://google.com",
			contentType: "text/plain",
			storage: &fakeStorage{
				searchResult: st.ShortURL("abc123"),
				searchErr:    nil,
			},
			want: want{
				statusCode:  http.StatusCreated,
				contentType: "text/plain",
				bodyCheck: func(t *testing.T, body string) {
					expected := createURL(baseURL, st.ShortURL("abc123"))
					if body != expected {
						t.Fatalf("expected body %q, got %q", expected, body)
					}
				},
				searchCalls: 1,
				addCalls:    0,
			},
		},
		{
			name:        "wrong content type",
			method:      http.MethodPost,
			body:        "https://google.com",
			contentType: "application/json",
			storage:     &fakeStorage{},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "",
				bodyCheck: func(t *testing.T, body string) {
					if body != "" {
						t.Fatalf("expected empty body, got %q", body)
					}
				},
				searchCalls: 0,
				addCalls:    0,
			},
		},
		{
			name:        "empty body",
			method:      http.MethodPost,
			body:        "",
			contentType: "text/plain",
			storage: &fakeStorage{},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain",
				bodyCheck: nil,
				searchCalls: 0,
				addCalls:    0,
			},
		},
		{
			name:        "bad body 1",
			method:      http.MethodPost,
			body:        "a,jshda\naslkjdgh7162//\\",
			contentType: "text/plain",
			storage: &fakeStorage{},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain",
				bodyCheck: nil,
				searchCalls: 0,
				addCalls:    0,
			},
		},
		{
			name:        "bad body 2",
			method:      http.MethodPost,
			body:        "\n\r////",
			contentType: "text/plain",
			storage: &fakeStorage{},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain",
				bodyCheck: nil,
				searchCalls: 0,
				addCalls:    0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			rr := httptest.NewRecorder()

			h := ShortenLinkHandler(tt.storage, baseURL)
			h.ServeHTTP(rr, req)

			if rr.Code != tt.want.statusCode {
				t.Errorf("expected status %d, got %d", tt.want.statusCode, rr.Code)
			}

			if tt.want.contentType != "" {
				ct := rr.Header().Get("Content-Type")
				if ct != tt.want.contentType {
					t.Errorf("expected Content-Type %q, got %q", tt.want.contentType, ct)
				}
			}

			if tt.want.bodyCheck != nil {
				tt.want.bodyCheck(t, rr.Body.String())
			}

			if tt.storage.searchCalls != tt.want.searchCalls {
				t.Errorf("expected Search calls %d, got %d", tt.want.searchCalls, tt.storage.searchCalls)
			}
			if tt.storage.addCalls != tt.want.addCalls {
				t.Errorf("expected Add calls %d, got %d", tt.want.addCalls, tt.storage.addCalls)
			}
		})
	}
}
