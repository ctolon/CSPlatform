package service

import (
	"a0/internal/config"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
)

type MetricsService struct {
	log zerolog.Logger
	config *config.Config
}

func NewMetricsService(log zerolog.Logger, config *config.Config) *MetricsService {
	return &MetricsService{log, config}
}

func (s *MetricsService) GetCPUUsage(prevIdle, prevTotal uint64) (float64, uint64, uint64) {
	path := fmt.Sprintf("%s/stat", s.config.Server.ProcPath)
	data, err := os.ReadFile(path)
	if err != nil {
		s.log.Println("Error reading /proc/stat:", err)
		return 0, prevIdle, prevTotal
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		if fields[0] == "cpu" {
			var total, idle uint64
			for i, v := range fields[1:] {
				val, _ := strconv.ParseUint(v, 10, 64)
				total += val
				if i == 3 {
					idle = val
				}
			}
			var usage float64
			deltaTotal := total - prevTotal
			deltaIdle := idle - prevIdle
			if deltaTotal > 0 {
				usage = float64(deltaTotal-deltaIdle) / float64(deltaTotal) * 100
			}
			return usage, idle, total
		}
	}
	return 0, prevIdle, prevTotal
}

func (s *MetricsService) GetRAMUsage() float64 {
	path := fmt.Sprintf("%s/meminfo", s.config.Server.ProcPath)
	data, err := os.ReadFile(path)
	if err != nil {
		s.log.Println("Error reading /proc/meminfo:", err)
		return 0
	}

	var total, free, buffers, cached uint64
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		val, _ := strconv.ParseUint(fields[1], 10, 64)
		switch key {
		case "MemTotal":
			total = val
		case "MemFree":
			free = val
		case "Buffers":
			buffers = val
		case "Cached":
			cached = val
		}
	}

	if total == 0 {
		return 0
	}
	used := total - free - buffers - cached
	return float64(used) / float64(total) * 100
}

func (s *MetricsService) GetRAMUsageR() float64 {
	path := fmt.Sprintf("%s/meminfo", s.config.Server.ProcPath)
	data, err := os.ReadFile(path)
	if err != nil {
		s.log.Println("Error reading /proc/meminfo:", err)
		return 0
	}

	var total, available uint64
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		val, _ := strconv.ParseUint(fields[1], 10, 64)
		switch key {
		case "MemTotal":
			total = val
		case "MemAvailable":
			available = val
		}
	}

	if total == 0 {
		return 0
	}

	used := total - available
	return float64(used) / float64(total) * 100
}
