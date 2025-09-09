package handlers

import (
	"a0/internal/app/service"
	"github.com/labstack/echo/v4"
	"net/http"
)

// Handler struct
type ContainerHandler struct {
	Service *service.ContainerService
}

func NewContainerHandler(s *service.ContainerService) *ContainerHandler {
	return &ContainerHandler{Service: s}
}

func (h *ContainerHandler) CreateContainer(c echo.Context) error {
	req := new(service.CreateContainerRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	resp, err := h.Service.CreateContainer(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, resp)
}

func (h *ContainerHandler) StartContainer(c echo.Context) error {
	id := c.Param("id")
	if err := h.Service.StartContainer(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "started"})
}

func (h *ContainerHandler) StopContainer(c echo.Context) error {
	id := c.Param("id")
	if err := h.Service.StopContainer(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "stopped"})
}

func (h *ContainerHandler) RestartContainer(c echo.Context) error {
	id := c.Param("id")
	if err := h.Service.RestartContainer(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "restarted"})
}

func (h *ContainerHandler) RemoveContainer(c echo.Context) error {
	id := c.Param("id")
	force := c.QueryParam("force") == "true"
	if err := h.Service.RemoveContainer(id, force); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "removed"})
}

func (h *ContainerHandler) ListContainers(c echo.Context) error {
	all := c.QueryParam("all") == "true"
	containers, err := h.Service.ListContainers(all)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, containers)
}

func (h *ContainerHandler) ListCodeServerContainers(c echo.Context) error {
	containers, err := h.Service.ListCodeServerContainersPrefix()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, containers)
}

func (h *ContainerHandler) LogsContainer(c echo.Context) error {
	id := c.Param("id")
	tail := c.QueryParam("tail")
	logs, err := h.Service.LogsContainer(id, tail)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"logs": logs})
}

func (h *ContainerHandler) GetContainerIDByName(c echo.Context) error {
	name := c.Param("name")
	id, err := h.Service.GetContainerIDByName(name)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"id": id})
}

func (h *ContainerHandler) GetConfigDefaultsHandler(c echo.Context) error {
	resp, err := h.Service.GetConfigDefaults()
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *ContainerHandler) IsContainerExistHandler(c echo.Context) error {
	name := c.Param("name")
	exist, err := h.Service.IsContainerExist(name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"name":  name,
		"exist": exist,
	})
}

// IsContainerRunningHandler checks if a container is currently running
func (h *ContainerHandler) IsContainerRunningHandler(c echo.Context) error {
	name := c.Param("name")
	running, err := h.Service.IsContainerRunning(name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"name":    name,
		"running": running,
	})
}
