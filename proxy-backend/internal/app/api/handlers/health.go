package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type HealthCheckHandler struct{}

func NewHealthCheckHandler() *HealthCheckHandler {
	return &HealthCheckHandler{}
}

func (h *HealthCheckHandler) Healthcheck(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}
