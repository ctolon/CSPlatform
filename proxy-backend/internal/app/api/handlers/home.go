package handlers

import (
	"context"
	"html/template"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"v0/internal/app/security"
	"v0/internal/app/service"
	"v0/internal/config"
)

type HomePageHandler struct {
	jwtService   *security.JWTService
	tmpl         *template.Template
	config       *config.AppConfig
	log          zerolog.Logger
	agentService *service.AgentService
	reg          *service.ContainerRegistryService
}

func NewHomePageHandler(
	jwtService *security.JWTService,
	tmpl *template.Template,
	config *config.AppConfig,
	log zerolog.Logger,
	agentService *service.AgentService,
	reg *service.ContainerRegistryService,
) *HomePageHandler {
	return &HomePageHandler{jwtService, tmpl, config, log, agentService, reg}
}

func (h *HomePageHandler) RenderHomePage(c echo.Context) error {
	data := make(map[string]any)
	data["Username"] = c.Get("username").(string)
	data["CSRFToken"] = c.Get("csrf").(string)
	data["AdminGroup"] = h.config.AuthAdminRoles
	groupsFromCtx := c.Get("groups")
	if groups, err := h.jwtService.ClaimToStringSlice(groupsFromCtx); err == nil {
		data["Groups"] = groups
	}

	// check has container and is running
	ctx := context.Background()
	if resp, err := h.reg.Get(ctx, data["Username"].(string)); err == nil {
		data["HasContainer"] = true
		data["AgentHost"] = resp.AgentHost
		data["CreatedAt"] = resp.CreatedAt
		if resp, err := h.agentService.IsContainerRunning(resp.AgentHost, resp.ContainerName); err == nil {
			data["IsContainerRunning"] = resp.Running
		}

	} else if err.Error() == "container not found" {
		data["HasContainer"] = false
		data["IsContainerRunning"] = false
	} else {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return h.tmpl.ExecuteTemplate(c.Response(), "home.go.tmpl", data)
}
