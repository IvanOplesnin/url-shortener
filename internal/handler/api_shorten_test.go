package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IvanOplesnin/url-shortener/internal/model"
	repo "github.com/IvanOplesnin/url-shortener/internal/repository"
	mock_repo "github.com/IvanOplesnin/url-shortener/internal/repository/mock_repo"
	"github.com/IvanOplesnin/url-shortener/internal/service/shortener"
	u "github.com/IvanOplesnin/url-shortener/internal/service/url"
	"go.uber.org/mock/gomock"
)

func TestShortenApiHandler(t *testing.T) {
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
		setupMock   func(m *mock_repo.MockRepo)
		want        want
	}{
		{
			name:        "new link in empty storage (json)",
			method:      http.MethodPost,
			body:        []byte(`{"url":"https://google.com"}`),
			contentType: applicationJSONValue,
			setupMock: func(m *mock_repo.MockRepo) {
				m.EXPECT().
					Search(repo.URL("https://google.com")).
					Return(repo.ShortURL(""), repo.ErrNotFoundURL).
					Times(1)

				m.EXPECT().
					Add(gomock.Any(), repo.URL("https://google.com")).
					Return(nil).
					Times(1)
			},
			want: want{
				statusCode:  http.StatusCreated,
				contentType: applicationJSONValue,
				bodyCheck: func(t *testing.T, body []byte) {
					t.Helper()
					var res model.ResponseBody
					if err := json.Unmarshal(body, &res); err != nil {
						t.Fatalf("invalid json body %q: %v", string(body), err)
					}
					if res.Result == "" {
						t.Fatalf("expected non-empty result, got empty")
					}
					if !strings.HasPrefix(res.Result, baseURL+"/") {
						t.Fatalf("expected result to start with %q, got %q", baseURL+"/", res.Result)
					}
				},
			},
		},

		// 2. Ссылка уже существует в хранилище
		{
			name:        "existing link in storage (json)",
			method:      http.MethodPost,
			body:        []byte(`{"url":"https://google.com"}`),
			contentType: applicationJSONValue,
			setupMock: func(m *mock_repo.MockRepo) {
				m.EXPECT().
					Search(repo.URL("https://google.com")).
					Return(repo.ShortURL("abc123"), nil).
					Times(1)

				m.EXPECT().
					Add(gomock.Any(), gomock.Any()).
					Times(0)
			},
			want: want{
				statusCode:  http.StatusCreated, // если переделаешь логику на 409, поменяй здесь
				contentType: applicationJSONValue,
				bodyCheck: func(t *testing.T, body []byte) {
					t.Helper()
					var res model.ResponseBody
					if err := json.Unmarshal(body, &res); err != nil {
						t.Fatalf("invalid json body %q: %v", string(body), err)
					}
					expected, _ := u.CreateURL(baseURL, repo.ShortURL("abc123"))
					if res.Result != expected {
						t.Fatalf("expected result %q, got %q", expected, res.Result)
					}
				},
			},
		},

		// 3. Неверный Content-Type
		{
			name:        "wrong content type (json handler)",
			method:      http.MethodPost,
			body:        []byte(`{"url":"https://google.com"}`),
			contentType: "text/plain", // не application/json
			setupMock: func(m *mock_repo.MockRepo) {
				m.EXPECT().Search(gomock.Any()).Times(0)
				m.EXPECT().Add(gomock.Any(), gomock.Any()).Times(0)
			},
			want: want{
				// здесь заложим желаемое поведение: 400 и пустое тело
				// текущий хендлер ничего не пишет вообще → тест как раз подсветит это
				statusCode:  http.StatusBadRequest,
				contentType: "",
				bodyCheck: func(t *testing.T, body []byte) {
					t.Helper()
					if len(body) != 0 {
						t.Fatalf("expected empty body, got %q", string(body))
					}
				},
			},
		},

		// 4. Битый JSON
		{
			name:        "invalid json body",
			method:      http.MethodPost,
			body:        []byte(`{"url":"https://google.com"`), // нет закрывающей }
			contentType: applicationJSONValue,
			setupMock: func(m *mock_repo.MockRepo) {
				m.EXPECT().Search(gomock.Any()).Times(0)
				m.EXPECT().Add(gomock.Any(), gomock.Any()).Times(0)
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: applicationJSONValue,
				bodyCheck: func(t *testing.T, body []byte) {
					t.Helper()
					if len(body) != 0 {
						t.Fatalf("expected empty body on bad json, got %q", string(body))
					}
				},
			},
		},

		// 5. JSON без поля url
		{
			name:        "missing url field",
			method:      http.MethodPost,
			body:        []byte(`{}`),
			contentType: applicationJSONValue,
			setupMock: func(m *mock_repo.MockRepo) {
				m.EXPECT().Search(gomock.Any()).Times(0)
				m.EXPECT().Add(gomock.Any(), gomock.Any()).Times(0)
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: applicationJSONValue,
				bodyCheck:   nil,
			},
		},

		// 6. Невалидный URL в поле url
		{
			name:        "invalid url value",
			method:      http.MethodPost,
			body:        []byte(`{"url":"not\a\valid-url"}`),
			contentType: applicationJSONValue,
			setupMock: func(m *mock_repo.MockRepo) {
				m.EXPECT().Search(gomock.Any()).Times(0)
				m.EXPECT().Add(gomock.Any(), gomock.Any()).Times(0)
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: applicationJSONValue,
				bodyCheck:   nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repository := mock_repo.NewMockStorage(ctrl)

			if tt.setupMock != nil {
				tt.setupMock(repository)
			}

			req := httptest.NewRequest(tt.method, "/api/shorten", bytes.NewReader(tt.body))

			if tt.contentType != "" {
				req.Header.Set(contentTypeKey, tt.contentType)
			}

			rr := httptest.NewRecorder()
			
			newService := shortener.New(repository, baseURL)

			h := ShortenAPIHandler(newService)

			h.ServeHTTP(rr, req)

			// Проверка: совпадает ли код ответа с ожидаемым
			if rr.Code != tt.want.statusCode {
				t.Errorf("expected status %d, got %d", tt.want.statusCode, rr.Code)
				return
			}

			// Проверка заголовка Content-Type, если он ожидается
			if tt.want.contentType != "" {
				ct := rr.Header().Get(contentTypeKey)
				if ct != tt.want.contentType {
					t.Errorf("expected Content-Type %q, got %q", tt.want.contentType, ct)
					return
				}
			}

			// Проверка тела ответа, если задана функция bodyCheck
			if tt.want.bodyCheck != nil {
				tt.want.bodyCheck(t, rr.Body.Bytes())
			}

		})
	}
}
