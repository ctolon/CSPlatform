package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"a0/internal/app/service"
)

type MetricsResponse struct {
	CPU    float64 `json:"cpu_percent"`
	CPUStr string  `json:"cpu_percent_str"`
	RAM    float64 `json:"ram_percent"`
	RAMStr string  `json:"ram_percent_str"`
	Idle   uint64  `json:"idle"`
	Total  uint64  `json:"total"`
}

type MetricsHandler struct {
	s *service.MetricsService
}

func NewMetricsHandler(s *service.MetricsService) *MetricsHandler {
	return &MetricsHandler{s}
}

func (h *MetricsHandler) Fetch(c echo.Context) error {
	var prevIdle, prevTotal uint64
	cpu, idle, total := h.s.GetCPUUsage(prevIdle, prevTotal)
	prevIdle = idle
	prevTotal = total

	ram := h.s.GetRAMUsageR()

	resp := MetricsResponse{
		CPU:    cpu,
		CPUStr: strconv.FormatFloat(cpu, 'f', 2, 64) + "%",
		RAM:    ram,
		RAMStr: strconv.FormatFloat(ram, 'f', 2, 64) + "%",
		Idle:   idle,
		Total:  total,
	}
	return c.JSON(http.StatusOK, resp)
}
