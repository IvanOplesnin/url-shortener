package handlers

import (
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

	getResult st.URL
	getErr    error

	searchCalls int
	addCalls    int
	getCalls    int

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
	f.getCalls++
	return f.getResult, f.getErr
}

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
		searchCalls int                             // Сколько раз должен быть вызван Search
		addCalls    int                             // Сколько раз должен быть вызван Add
	}

	// Определение тестовых сценариев
	tests := []struct {
		name        string       // Название теста — описывает сценарий
		method      string       // HTTP-метод запроса
		body        string       // Тело запроса (исходная ссылка или некорректные данные)
		contentType string       // Content-Type запроса
		storage     *fakeStorage // Мок-хранилище с предустановленным поведением
		want        want         // Ожидаемые результаты
	}{
		// Сценарий 1: новая ссылка, отсутствует в хранилище
		{
			name:        "new link in empty storage",
			method:      http.MethodPost,
			body:        "https://google.com",
			contentType: "text/plain",
			storage: &fakeStorage{
				searchErr: st.ErrNotFoundURL, // Имитация: ссылка не найдена при поиске
				addErr:    nil,               // Добавление проходит успешно
				getErr:    st.ErrNotFoundShortURL,
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
				searchCalls: 1, // Search должен быть вызван один раз
				addCalls:    1, // Add должен быть вызван для сохранения новой ссылки
			},
		},
		// Сценарий 2: ссылка уже существует в хранилище
		{
			name:        "existing link in storage",
			method:      http.MethodPost,
			body:        "https://google.com",
			contentType: "text/plain",
			storage: &fakeStorage{
				searchResult: st.ShortURL("abc123"), // Возвращается существующий ID
				searchErr:    nil,                   // Поиск успешен
			},
			want: want{
				statusCode:  http.StatusCreated,
				contentType: "text/plain",
				bodyCheck: func(t *testing.T, body string) {
					// Ожидается полный URL: baseURL + "/abc123"
					expected, _ := createURL(baseURL, st.ShortURL("abc123"))
					if body != expected {
						t.Fatalf("expected body %q, got %q", expected, body)
					}
				},
				searchCalls: 1, // Search вызван один раз
				addCalls:    0, // Add не вызывается — дублирование не требуется
			},
		},
		// Сценарий 3: неверный тип содержимого
		{
			name:        "wrong content type",
			method:      http.MethodPost,
			body:        "https://google.com",
			contentType: "application/json", // Неподдерживаемый Content-Type
			storage:     &fakeStorage{},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "",
				bodyCheck: func(t *testing.T, body string) {
					// Ожидается пустое тело при ошибке
					if body != "" {
						t.Fatalf("expected empty body, got %q", body)
					}
				},
				searchCalls: 0, // Search не должен вызываться
				addCalls:    0, // Add не должен вызываться
			},
		},
		// Сценарий 4: пустое тело запроса
		{
			name:        "empty body",
			method:      http.MethodPost,
			body:        "",
			contentType: "text/plain",
			storage:     &fakeStorage{},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain",
				bodyCheck:   nil, // Проверка тела не требуется — достаточно статуса
				searchCalls: 0,
				addCalls:    0,
			},
		},
		// Сценарий 5: тело с битыми символами (не URL)
		{
			name:        "bad body 1",
			method:      http.MethodPost,
			body:        "a,jshda\naslkjdgh7162//\\",
			contentType: "text/plain",
			storage:     &fakeStorage{},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain",
				bodyCheck:   nil,
				searchCalls: 0,
				addCalls:    0,
			},
		},
		// Сценарий 6: тело с невалидными символами
		{
			name:        "bad body 2",
			method:      http.MethodPost,
			body:        "\n\r////",
			contentType: "text/plain",
			storage:     &fakeStorage{},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain",
				bodyCheck:   nil,
				searchCalls: 0,
				addCalls:    0,
			},
		},
	}

	// Выполнение каждого тестового случая
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создание HTTP-запроса с заданным методом и телом
			req := httptest.NewRequest(tt.method, "/", strings.NewReader(tt.body))
			// Установка заголовка Content-Type, если он задан
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			// Запись ответа
			rr := httptest.NewRecorder()

			// Инициализация обработчика с моком хранилища и базовым URL
			h := ShortenLinkHandler(tt.storage, baseURL)
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

			// Проверка количества вызовов метода Search хранилища
			if tt.storage.searchCalls != tt.want.searchCalls {
				t.Errorf("expected Search calls %d, got %d", tt.want.searchCalls, tt.storage.searchCalls)
				return
			}

			// Проверка количества вызовов метода Add хранилища
			if tt.storage.addCalls != tt.want.addCalls {
				t.Errorf("expected Add calls %d, got %d", tt.want.addCalls, tt.storage.addCalls)
				return
			}
		})
	}
}
