package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IvanOplesnin/url-shortener/internal/model"
	"github.com/IvanOplesnin/url-shortener/internal/repository"
	"github.com/IvanOplesnin/url-shortener/internal/service/shortener"
	mock_repo "github.com/IvanOplesnin/url-shortener/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestBatchApiShortenHandler(t *testing.T) {
	baseURL := "http://localhost:8080"

	type want struct {
		statusCode  int
		contentType string
		bodyCheck   func(t *testing.T, body []byte)
	}

	tests := []struct {
		name        string
		method      string
		body        []byte
		contentType string
		setupMock   func(m *mock_repo.MockBatchRepo)
		want        want
	}{
		{
			name:   "new links in empty storage (json)",
			method: http.MethodPost,
			body: []byte(`[{"correlation_id": "req-1", "original_url": "https://github.com"},
			{"correlation_id": "req-2", "original_url": "https://google.com"}]`),
			contentType: applicationJSONValue,
			setupMock: func(m *mock_repo.MockBatchRepo) {
				m.EXPECT().
					GetByURLs(gomock.Any(), []string{"https://github.com", "https://google.com"}).
					Return([]repository.Record{}, nil).
					Times(1)

				m.EXPECT().
					AddMany(gomock.Any(), gomock.Any()).
					Return([]repository.Record{
						{URL: repository.URL("https://github.com"), ShortURL: repository.ShortURL("AbCdE1")},
						{URL: repository.URL("https://google.com"), ShortURL: repository.ShortURL("ZxYwV2")},
					}, nil).
					Times(1)
			},
			want: want{
				statusCode:  http.StatusCreated,
				contentType: applicationJSONValue,
				bodyCheck: func(t *testing.T, body []byte) {
					t.Helper()
					var got []model.ResponseBatchBody
					require.NoError(t, json.Unmarshal(body, &got))
					require.Len(t, got, 2)

					require.Equal(t, "req-1", got[0].CorrelationID)
					require.True(t, strings.HasPrefix(got[0].ShortURL, baseURL+"/"))

					require.Equal(t, "req-2", got[1].CorrelationID)
					require.True(t, strings.HasPrefix(got[1].ShortURL, baseURL+"/"))
				},
			},
		},
		{
			name:   "one url exists, one new -> conflict",
			method: http.MethodPost,
			body: []byte(`[
			  {"correlation_id":"req-1","original_url":"https://github.com"},
			  {"correlation_id":"req-2","original_url":"https://google.com"}
			]`),
			contentType: applicationJSONValue,
			setupMock: func(m *mock_repo.MockBatchRepo) {
				// первый вызов GetByURLs по двум URL
				m.EXPECT().
					GetByURLs(gomock.Any(), []string{"https://github.com", "https://google.com"}).
					Return([]repository.Record{
						{URL: repository.URL("https://github.com"), ShortURL: repository.ShortURL("AbCdE1")},
					}, nil).
					Times(1)
				m.EXPECT().
					AddMany(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, args []repository.ArgAddMany) ([]repository.Record, error) {
						require.Len(t, args, 1)
						require.Equal(t, repository.URL("https://google.com"), args[0].URL)
						return []repository.Record{
							{URL: args[0].URL, ShortURL: args[0].ShortURL},
						}, nil
					}).
					Times(1)
			},
			want: want{
				statusCode:  http.StatusConflict,
				contentType: applicationJSONValue,
				bodyCheck: func(t *testing.T, body []byte) {
					t.Helper()
					var got []model.ResponseBatchBody
					require.NoError(t, json.Unmarshal(body, &got))
					require.Len(t, got, 2)

					// порядок должен совпасть с входом
					require.Equal(t, "req-1", got[0].CorrelationID)
					require.True(t, strings.HasPrefix(got[0].ShortURL, baseURL+"/"))

					require.Equal(t, "req-2", got[1].CorrelationID)
					require.True(t, strings.HasPrefix(got[1].ShortURL, baseURL+"/"))
				},
			},
		},
		{
			name:        "invalid json",
			method:      http.MethodPost,
			body:        []byte(`[{"correlation_id":"req-1","original_url":"https://example.com",}]`), // лишняя запятая
			contentType: applicationJSONValue,
			setupMock: func(m *mock_repo.MockBatchRepo) {
				// не должен трогать repo вообще
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: applicationJSONValue,
				bodyCheck:   nil,
			},
		},
		{
			name:        "wrong content-type",
			method:      http.MethodPost,
			body:        []byte(`[]`),
			contentType: "text/plain",
			setupMock: func(m *mock_repo.MockBatchRepo) {
				// repo не вызывается
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "",
			},
		},
		{
			name:   "duplicate original_url in request",
			method: http.MethodPost,
			body: []byte(`[
			  {"correlation_id":"req-1","original_url":"https://github.com"},
			  {"correlation_id":"req-2","original_url":"https://github.com"}
			]`),
			contentType: applicationJSONValue,
			setupMock: func(m *mock_repo.MockBatchRepo) {
				// repo не должен вызываться, т.к. ошибка до транзакции
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: applicationJSONValue,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mock_repo.NewMockBatchRepo(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			req := httptest.NewRequest(tt.method, "/api/shorten/batch", bytes.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set(contentTypeKey, tt.contentType)
			}

			rr := httptest.NewRecorder()

			svc := shortener.New(repo, baseURL)
			h := ShortenBatchAPIHandler(svc)

			h.ServeHTTP(rr, req)

			// Проверка: совпадает ли код ответа с ожидаемым
			if rr.Code != tt.want.statusCode {
				t.Errorf("expected status %d, got %d", tt.want.statusCode, rr.Code)
				return
			}

			// Проверка заголовка Content-Type, если он ожидается
			ct := rr.Header().Get(contentTypeKey)
			if ct != tt.want.contentType {
				t.Errorf("expected Content-Type %q, got %q", tt.want.contentType, ct)
				return
			}

			// Проверка тела ответа, если задана функция bodyCheck
			if tt.want.bodyCheck != nil {
				tt.want.bodyCheck(t, rr.Body.Bytes())
			}

		})
	}

}
