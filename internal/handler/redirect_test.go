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

// TestRedirectHandler тестирует поведение HTTP-обработчика редиректа по короткому идентификатору.
//
// Тест проверяет корректность ответов сервера при переходе по короткой ссылке.
// Используется мок-хранилище (fakeRedirectStorage), чтобы изолировать логику обработчика
// от реальной реализации хранилища и протестировать разные сценарии поведения.
//
// Каждый подтест проверяет:
//   - Корректность HTTP-статуса
//   - Наличие и значение заголовка Location при редиректе
//   - Количество вызовов метода Get хранилища
//   - Правильность переданного идентификатора в метод Get
func TestRedirectHandler(t *testing.T) {
	// Базовый URL сервера — имитирует реальный адрес, по которому работает сервис
	baseURL := "http://localhost:8080"

	// Структура want определяет ожидаемые значения для каждого тестового случая
	type want struct {
		statusCode int    // Ожидаемый HTTP-статус ответа
		Location   string // Ожидаемое значение заголовка Location (если должно быть)
		getCalls   int    // Сколько раз должен быть вызван метод Get хранилища
		getID      string // Какой ID должен быть передан в метод Get
	}

	// Определение тестовых случаев
	tests := []struct {
		name    string               // Название теста — описывает сценарий
		method  string               // HTTP-метод запроса
		path    string               // Путь запроса (включая базовый URL)
		storage *fakeRedirectStorage // Мок-хранилище с предустановленным поведением
		want    want                 // Ожидаемые результаты
	}{
		// Тест 1: успешный редирект
		{
			name:   "success redirect",  // Описание: успешный переход по короткой ссылке
			method: http.MethodGet,      // Ожидается GET-запрос
			path:   baseURL + "/abc123", // Запрос по адресу вида http://localhost:8080/abc123
			storage: &fakeRedirectStorage{
				getURL: "https://google.com", // Мок возвращает целевой URL
			},
			want: want{
				statusCode: http.StatusTemporaryRedirect, // Ожидается временный редирект (307)
				Location:   "https://google.com",         // Заголовок Location должен указывать на целевой URL
				getCalls:   1,                            // Метод Get хранилища должен быть вызван один раз
				getID:      "abc123",                     // В Get должен быть передан ID "abc123"
			},
		},
		// Тест 2: ссылка не найдена
		{
			name:   "not found",         // Описание: запрашиваемая короткая ссылка отсутствует
			method: http.MethodGet,      // GET-запрос
			path:   baseURL + "/abc123", // Аналогичный путь
			storage: &fakeRedirectStorage{
				getErr: st.ErrNotFoundShortURL, // Мок возвращает ошибку "не найдено"
			},
			want: want{
				statusCode: http.StatusNotFound, // Ожидается статус 404
				getCalls:   1,                   // Get вызван один раз
				getID:      "abc123",            // ID передан корректно
				// Location не ожидается, так как редирект не происходит
			},
		},
	}

	// Запуск каждого тестового случая
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создание HTTP-запроса с заданным методом и путём
			req := httptest.NewRequest(tt.method, tt.path, nil)
			// Создание записи ответа (ResponseRecorder)
			rr := httptest.NewRecorder()

			// Инициализация маршрутизатора с мок-хранилищем и базовым URL
			// InitHandlers настраивает маршруты, включая обработчик редиректа
			mux := InitHandlers(tt.storage, baseURL)
			// Обработка запроса
			mux.ServeHTTP(rr, req)

			// Проверка: совпадает ли код статуса с ожидаемым
			if rr.Code != tt.want.statusCode {
				t.Errorf("expected status %d, got %d", tt.want.statusCode, rr.Code)
			}

			// Проверка заголовка Location, если он ожидается
			if tt.want.Location != "" {
				loc := rr.Header().Get("Location")
				if loc != tt.want.Location {
					t.Errorf("expected Location %q, got %q", tt.want.Location, loc)
				}
			}

			// Проверка количества вызовов метода Get хранилища
			if tt.want.getCalls != tt.storage.getCalls {
				t.Errorf(
					"expected Get calls %d, got %d",
					tt.want.getCalls,
					tt.storage.getCalls,
				)
			}

			// Проверка, что в метод Get был передан правильный ID
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
