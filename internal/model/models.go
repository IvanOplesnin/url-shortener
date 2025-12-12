package model

import storage "github.com/IvanOplesnin/url-shortener/internal/repository"

type RequestBody struct {
	Url storage.URL `json:"url"`
}

type ResponseBody struct {
	Result string `json:"result"`
}
