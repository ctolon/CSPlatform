package handlers

import (
	"a0/internal/config"
	"net/http"

	"github.com/labstack/echo/v4"
)


type AgentHandler struct {
	config *config.Config
}

func NewAgentHandler(config *config.Config) *AgentHandler {
	return &AgentHandler{config}
}

func (h *AgentHandler) GetTags(c echo.Context) error {
	return c.JSON(http.StatusOK, h.config.AgentMetadata.Tags)
}