package handlers

import (
	"html/template"

	"github.com/labstack/echo/v4"
)

type NotFoundPageHandler struct {
	tmpl *template.Template
}

func NewNotFoundPageHandler(tmpl *template.Template) *NotFoundPageHandler {
	return &NotFoundPageHandler{tmpl}
}

func (h *NotFoundPageHandler) Render404(c echo.Context) error {
	isLoggedIn := false
	data := make(map[string]any)
	data["RequestID"] = c.Response().Header().Get(echo.HeaderXRequestID)
	val := c.Get("username")
	username, ok := val.(string)
	if ok && username != "" {
		isLoggedIn = true
	}
	data["IsLoggedIn"] = isLoggedIn
	data["FromProxy"] = false
	data["Details"] = "page not found"

	return h.tmpl.ExecuteTemplate(c.Response(), "404.go.tmpl", data)
}
