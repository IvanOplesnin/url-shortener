package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	repo "github.com/IvanOplesnin/url-shortener/internal/repository"
	mock_repo "github.com/IvanOplesnin/url-shortener/internal/repository/mock_repo"
	"go.uber.org/mock/gomock"
)

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
	}

	// Определение тестовых случаев
	tests := []struct {
		name      string                    // Название теста — описывает сценарий
		method    string                    // HTTP-метод запроса
		path      string                    // Путь запроса (включая базовый URL)
		setupMock func(*mock_repo.MockRepo) // Мок-хранилище с предустановленным поведением
		want      want                      // Ожидаемые результаты
	}{
		// Тест 1: успешный редирект
		{
			name:   "success redirect",  // Описание: успешный переход по короткой ссылке
			method: http.MethodGet,      // Ожидается GET-запрос
			path:   baseURL + "/abc123", // Запрос по адресу вида http://localhost:8080/abc123
			setupMock: func(m *mock_repo.MockRepo) {
				m.EXPECT().Get(repo.ShortURL("abc123")).Return(repo.URL("https://google.com"), nil).Times(1)
			},
			want: want{
				statusCode: http.StatusTemporaryRedirect, // Ожидается временный редирект (307)
				Location:   "https://google.com",         // Заголовок Location должен указывать на целевой URL
			},
		},
		// Тест 2: ссылка не найдена
		{
			name:   "not found",         // Описание: запрашиваемая короткая ссылка отсутствует
			method: http.MethodGet,      // GET-запрос
			path:   baseURL + "/abc123", // Аналогичный путь
			setupMock: func(m *mock_repo.MockRepo) {
				m.EXPECT().Get(repo.ShortURL("abc123")).Return(repo.URL(""), repo.ErrNotFoundURL).Times(1)
			},
			want: want{
				statusCode: http.StatusNotFound, // Ожидается статус 404
				// Location не ожидается, так как редирект не происходит
			},
		},
	}

	// Запуск каждого тестового случая
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			storage := mock_repo.NewMockStorage(ctrl)

			if tt.setupMock != nil {
				tt.setupMock(storage)
			}

			// Создание HTTP-запроса с заданным методом и путём
			req := httptest.NewRequest(tt.method, tt.path, nil)
			// Создание записи ответа (ResponseRecorder)
			rr := httptest.NewRecorder()

			// Инициализация маршрутизатора с мок-хранилищем и базовым URL
			// InitHandlers настраивает маршруты, включая обработчик редиректа
			mux := InitHandlers(storage, baseURL)
			// Обработка запроса
			mux.ServeHTTP(rr, req)

			// Проверка: совпадает ли код статуса с ожидаемым
			if rr.Code != tt.want.statusCode {
				t.Errorf("expected status %d, got %d", tt.want.statusCode, rr.Code)
				return
			}

			// Проверка заголовка Location, если он ожидается
			if tt.want.Location != "" {
				loc := rr.Header().Get("Location")
				if loc != tt.want.Location {
					t.Errorf("expected Location %q, got %q", tt.want.Location, loc)
					return
				}
			}
		})
	}
}
