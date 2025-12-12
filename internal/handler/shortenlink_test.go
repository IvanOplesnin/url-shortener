package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	st "github.com/IvanOplesnin/url-shortener/internal/repository"
	mock_storage "github.com/IvanOplesnin/url-shortener/internal/repository/mock"
	u "github.com/IvanOplesnin/url-shortener/internal/service/url"
	"go.uber.org/mock/gomock"
)

// TestShortenLinkHandler тестирует HTTP-обработчик создания короткой ссылки.
//
// Проверяются различные сценарии:
//   - Успешное сокращение новой ссылки
//   - Повторное сокращение уже существующей ссылки (возврат существующего ID)
//   - Обработка ошибок: неверный Content-Type, пустое или некорректное тело
//
// Для изоляции логики используется мок-хранилище (*fakeStorage),
// позволяющее контролировать результаты методов Search и Add.
func TestShortenLinkHandler(t *testing.T) {
	// Базовый URL сервиса — используется для формирования полного короткого URL
	baseURL := "http://localhost:8080"

	// Структура want определяет ожидаемые результаты каждого тестового случая
	type want struct {
		statusCode  int                             // Ожидаемый HTTP-статус
		contentType string                          // Ожидаемый заголовок Content-Type
		bodyCheck   func(t *testing.T, body string) // Функция для проверки тела ответа
	}

	// Определение тестовых сценариев
	tests := []struct {
		name        string                            // Название теста — описывает сценарий
		method      string                            // HTTP-метод запроса
		body        string                            // Тело запроса (исходная ссылка или некорректные данные)
		contentType string                            // Content-Type запроса
		setupMock   func(m *mock_storage.MockStorage) //
		want        want                              // Ожидаемые результаты
	}{
		// Сценарий 1: новая ссылка, отсутствует в хранилище
		{
			name:        "new link in empty storage",
			method:      http.MethodPost,
			body:        "https://google.com",
			contentType: "text/plain",
			setupMock: func(m *mock_storage.MockStorage) {
				m.EXPECT().
					Search(st.URL("https://google.com")).
					Return(st.ShortURL(""), st.ErrNotFoundURL).
					Times(1)

				m.EXPECT().
					Add(gomock.Any(), st.URL("https://google.com")).
					Return(nil).
					Times(1)
			},
			want: want{
				statusCode:  http.StatusCreated,
				contentType: "text/plain",
				bodyCheck: func(t *testing.T, body string) {
					// Проверка: тело ответа начинается с базового URL
					if !strings.HasPrefix(body, baseURL+"/") {
						t.Fatalf("expected body to start with %q, got %q", baseURL+"/", body)
					}
				},
			},
		},
		// Сценарий 2: ссылка уже существует в хранилище
		{
			name:        "existing link in storage",
			method:      http.MethodPost,
			body:        "https://google.com",
			contentType: "text/plain",
			setupMock: func(m *mock_storage.MockStorage) {
				m.EXPECT().
					Search(st.URL("https://google.com")).
					Return(st.ShortURL("abc123"), nil).
					Times(1)

				m.EXPECT().
					Add(gomock.Any(), gomock.Any()).
					Times(0)
			},
			want: want{
				statusCode:  http.StatusCreated,
				contentType: "text/plain",
				bodyCheck: func(t *testing.T, body string) {
					// Ожидается полный URL: baseURL + "/abc123"
					expected, _ := u.CreateURL(baseURL, st.ShortURL("abc123"))
					if body != expected {
						t.Fatalf("expected body %q, got %q", expected, body)
					}
				},
			},
		},
		// Сценарий 3: неверный тип содержимого
		{
			name:        "wrong content type",
			method:      http.MethodPost,
			body:        "https://google.com",
			contentType: "application/json", // Неподдерживаемый Content-Type
			setupMock: func(m *mock_storage.MockStorage) {
				m.EXPECT().Add(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().Search(gomock.Any()).Times(0)
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "",
				bodyCheck: func(t *testing.T, body string) {
					// Ожидается пустое тело при ошибке
					if body != "" {
						t.Fatalf("expected empty body, got %q", body)
					}
				},
			},
		},
		// Сценарий 4: пустое тело запроса
		{
			name:        "empty body",
			method:      http.MethodPost,
			body:        "",
			contentType: "text/plain",
			setupMock: func(m *mock_storage.MockStorage) {
				m.EXPECT().Add(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().Search(gomock.Any()).Times(0)
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain",
				bodyCheck:   nil, // Проверка тела не требуется — достаточно статуса
			},
		},
		// Сценарий 5: тело с битыми символами (не URL)
		{
			name:        "bad body 1",
			method:      http.MethodPost,
			body:        "a,jshda\naslkjdgh7162//\\",
			contentType: "text/plain",
			setupMock: func(m *mock_storage.MockStorage) {
				m.EXPECT().Add(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().Search(gomock.Any()).Times(0)
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain",
				bodyCheck:   nil,
			},
		},
		// Сценарий 6: тело с невалидными символами
		{
			name:        "bad body 2",
			method:      http.MethodPost,
			body:        "\n\r////",
			contentType: "text/plain",
			setupMock:   nil,
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain",
				bodyCheck:   nil,
			},
		},
	}

	// Выполнение каждого тестового случая
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			storage := mock_storage.NewMockStorage(ctrl)

			if tt.setupMock != nil {
				tt.setupMock(storage)
			}

			// Создание HTTP-запроса с заданным методом и телом
			req := httptest.NewRequest(tt.method, "/", strings.NewReader(tt.body))
			// Установка заголовка Content-Type, если он задан
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			// Запись ответа
			rr := httptest.NewRecorder()

			// Инициализация обработчика с моком хранилища и базовым URL
			h := ShortenLinkHandler(storage, baseURL)
			// Обработка запроса
			h.ServeHTTP(rr, req)

			// Проверка: совпадает ли код ответа с ожидаемым
			if rr.Code != tt.want.statusCode {
				t.Errorf("expected status %d, got %d", tt.want.statusCode, rr.Code)
				return
			}

			// Проверка заголовка Content-Type, если он ожидается
			if tt.want.contentType != "" {
				ct := rr.Header().Get("Content-Type")
				if ct != tt.want.contentType {
					t.Errorf("expected Content-Type %q, got %q", tt.want.contentType, ct)
					return
				}
			}

			// Проверка тела ответа, если задана функция bodyCheck
			if tt.want.bodyCheck != nil {
				tt.want.bodyCheck(t, rr.Body.String())
			}
		})
	}
}
