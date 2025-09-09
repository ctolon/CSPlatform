package handlers

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"v0/internal/app/xdiscovery"
)

type DiscoveryHandler struct {
	registry *xdiscovery.Registry
	log      zerolog.Logger
}

func NewDiscoveryHandler(registry *xdiscovery.Registry, log zerolog.Logger) *DiscoveryHandler {
	return &DiscoveryHandler{registry, log}
}

func (h *DiscoveryHandler) Register(c echo.Context) error {
	var req struct {
		InstanceID    string            `json:"instanceID"`
		ServiceName   string            `json:"serviceName"`
		MainHost      string            `json:"mainHost"`
		MainHostProto string            `json:"mainHostProto"`
		HostPort      string            `json:"hostPort"`
		HostPortProto string            `mapstructure:"hostPortProto"`
		Version       string            `json:"version,omitempty"`
		Region        string            `json:"region,omitempty"`
		Tags          map[string]string `json:"tags,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	inst := xdiscovery.ServiceInstance{
		MainHost:      req.MainHost,
		MainHostProto: req.MainHostProto,
		HostPort:      req.HostPort,
		HostPortProto: req.HostPortProto,
		Version:       req.Version,
		Region:        req.Region,
		Tags:          req.Tags,
	}
	ctx := context.Background()
	h.log.Info().Msgf("Registering service with instanceID: %s", req.InstanceID)
	realIP := c.RealIP()
	if err := h.registry.Register(ctx, req.InstanceID, req.ServiceName, inst, realIP); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "registered"})
}

func (h *DiscoveryHandler) Deregister(c echo.Context) error {
	var req struct {
		InstanceID  string `json:"instanceID"`
		ServiceName string `json:"serviceName"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	ctx := context.Background()
	if err := h.registry.Deregister(ctx, req.InstanceID, req.ServiceName); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "deregistered"})
}

func (h *DiscoveryHandler) HealthCheck(c echo.Context) error {
	var req struct {
		InstanceID  string `json:"instanceID"`
		ServiceName string `json:"serviceName"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	ctx := context.Background()
	if err := h.registry.HealthCheck(ctx, req.InstanceID, req.ServiceName); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
}

func (h *DiscoveryHandler) Discover(c echo.Context) error {
	serviceName := c.Param("serviceName")
	ctx := context.Background()
	instances, err := h.registry.Discover(ctx, serviceName)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, instances)

}
