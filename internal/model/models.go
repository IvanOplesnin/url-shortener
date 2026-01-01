package model

import "github.com/IvanOplesnin/url-shortener/internal/repository"

type RequestBody struct {
	URL repository.URL `json:"url"`
}

type ResponseBody struct {
	Result string `json:"result"`
}

type RequestBatchBody struct {
	CorrelationID string         `json:"correlation_id"`
	OriginalURL   repository.URL `json:"original_url"`
}

type ResponseBatchBody struct {
	CorrelationID string              `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}
