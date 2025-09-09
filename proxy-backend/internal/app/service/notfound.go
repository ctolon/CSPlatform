package service

import (
	"html/template"

	"github.com/labstack/echo/v4"
)

type NotFoundPageService struct {
	tmpl *template.Template
}

func NewNotFoundPageService(tmpl *template.Template) *NotFoundPageService {
	return &NotFoundPageService{tmpl}
}

func (h *NotFoundPageService) Render404(c echo.Context, requestPath string, host string, details string, username string) error {
	isLoggedIn := false
	data := make(map[string]any)
	data["RequestID"] = c.Response().Header().Get(echo.HeaderXRequestID)
	data["IsLoggedIn"] = isLoggedIn
	data["RequestPath"] = requestPath
	if host != "" {
		data["Host"] = host
	}

	data["Details"] = details
	data["FromProxy"] = true
	return h.tmpl.ExecuteTemplate(c.Response(), "404.go.tmpl", data)
}
