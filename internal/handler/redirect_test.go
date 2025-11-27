package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	st "github.com/IvanOplesnin/url-shortener/internal/repository"
)

type fakeRedirectStorage struct {
	getURL   st.URL
	getErr   error
	getCalls int
	lastID   st.ShortURL
}

func (f *fakeRedirectStorage) Search(u st.URL) (st.ShortURL, error) {
	return "", errors.New("not used in this test")
}

func (f *fakeRedirectStorage) Add(id st.ShortURL, u st.URL) error {
	return errors.New("not used in this test")
}

func (f *fakeRedirectStorage) Get(id st.ShortURL) (st.URL, error) {
	f.getCalls++
	f.lastID = id
	return f.getURL, f.getErr
}

func TestRedirectHandler(t *testing.T) {
	baseURL := "http://localhost:8080"
	type want struct {
		statusCode int
		Location   string
		getCalls   int
		getID      string
	}
	tests := []struct {
		name    string
		method  string
		path    string
		storage *fakeRedirectStorage
		want    want
	}{
		{
			name:   "success redirect",
			method: http.MethodGet,
			path:   baseURL + "/abc123",
			storage: &fakeRedirectStorage{
				getURL: "https://google.com",
			},
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				Location:   "https://google.com",
				getCalls:   1,
				getID:      "abc123",
			},
		},
		{
			name:   "not found",
			method: http.MethodGet,
			path:   baseURL + "/abc123",
			storage: &fakeRedirectStorage{
				getErr: st.ErrNotFoundShortURL,
			},
			want: want{
				statusCode: http.StatusNotFound,
				getCalls:   1,
				getID:      "abc123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			mux := InitHandlers(tt.storage, baseURL)
			mux.ServeHTTP(rr, req)

			if rr.Code != tt.want.statusCode {
				t.Errorf("expected status %d, got %d", tt.want.statusCode, rr.Code)
			}
			
			if tt.want.Location != "" {
				loc := rr.Header().Get("Location")
				if loc != tt.want.Location {
					t.Errorf("expected Location %q, got %q", tt.want.Location, loc)
				}
			}
			if tt.want.getCalls != tt.storage.getCalls {
				t.Errorf(
					"expected Get calls %d, got %d",
					tt.want.getCalls,
					tt.storage.getCalls,
				)
			}

			if tt.want.getID != "" {
				if tt.want.getID != string(tt.storage.lastID) {
					t.Errorf(
						"expected Get ID %q, got %q",
						tt.want.getID,
						tt.storage.lastID,
					)
				}
			}
		})
	}

}
