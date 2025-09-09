package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"v0/internal/app/adapters"
)

type AgentServiceInfo struct {
	MainHost      string            `json:"mainHost"`
	MainHostProto string            `json:"mainHostProto"`
	HostPort      string            `json:"hostPort"`
	HostPortProto string            `json:"HostPortProto"`
	Version       string            `json:"version,omitempty"`
	Region        string            `json:"region,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
}

type IsContainerExistResponse struct {
	Name  string `json:"name"`
	Exist bool   `  json:"exist"`
}

type GetContainerIDByNameResponse struct {
	ID string `json:"id"`
}

type IsContainerRunningResponse struct {
	Name    string `json:"name"`
	Running bool   `  json:"running"`
}

type StartContainerResponse struct {
	Status string `json:"status"`
}

type StopContainerResponse struct {
	Status string `json:"status"`
}

type RestartContainerResponse struct {
	Status string `json:"status"`
}

type RemoveContainerResponse struct {
	Status string `json:"status"`
}

type CreateContainerResponse struct {
	ID       string   `json:"Id"`
	Warnings []string `json:"Warnings"`
}

type FetchMetricsResponse struct {
	CPU    float64 `json:"cpu_percent"`
	CPUStr string  `json:"cpu_percent_str"`
	RAM    float64 `json:"ram_percent"`
	RAMStr string  `json:"ram_percent_str"`
	Idle   uint64  `json:"idle"`
	Total  uint64  `json:"total"`
}

type GetContainerDefaultsResponse struct {
	Image      string            `json:"image"`
	Name       string            `json:"name"`
	Env        map[string]string `json:"env"`
	Network    string            `json:"network"`
	Volumes    []string          `json:"volumes"`
	Expose     []string          `json:"expose"`
	Ports      []string          `json:"ports"`
	CPUQuota   int64             `json:"cpuQuota"`
	Memory     string            `json:"memory"`
	Sysctls    map[string]string `json:"sysctls"`
	Restart    string            `json:"restart"`
	ExtraHosts []string          `json:"extra_hosts"`

	// Allow
	AllowEditImage      bool `json:"allowEditImage"`
	AllowEditName       bool `json:"allowEditName"`
	AllowEditMemory     bool `json:"allowEditMemory"`
	AllowEditCPU        bool `json:"allowEditCPU"`
	AllowEditRestart    bool `json:"allowEditRestart"`
	AllowEditNetwork    bool `json:"allowEditNetwork"`
	AllowEditPorts      bool `json:"allowEditPorts"`
	AllowEditExpose     bool `json:"allowEditExpose"`
	AllowEditVolumes    bool `json:"allowEditVolumes"`
	AllowEditExtraHosts bool `json:"allowEditExtraHosts"`
	AllowEditEnv        bool `json:"allowEditEnv"`
	AllowEditSysctls    bool `json:"allowEditSysctls"`
}

type AgentService struct {
	restyAdapter *adapters.RestyClientAdapter
	log          zerolog.Logger
	agentKey     string
	rdb          *redis.Client
}

func NewAgentService(restyAdapter *adapters.RestyClientAdapter, log zerolog.Logger, agentKey string, rdb *redis.Client) *AgentService {
	return &AgentService{restyAdapter, log, agentKey, rdb}
}

func (s *AgentService) IsContainerExist(agentURL string, containerName string) (*IsContainerExistResponse, error) {
	endpoint := fmt.Sprintf("/api/v1/containers/%s/exist", containerName)
	agentAPI := fmt.Sprintf("%s%s", agentURL, endpoint)
	resp, err := s.restyAdapter.R().
		SetHeader("Accept", "application/json").
		SetHeader("X-Agent-Key", s.agentKey).
		Get(agentAPI)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		var bodyStr string
		if resp.Body() != nil {
			bodyStr = string(resp.Body())
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode(), bodyStr)
	}
	var result IsContainerExistResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *AgentService) GetContainerIDByName(agentURL string, containerName string) (*GetContainerIDByNameResponse, error) {
	endpoint := fmt.Sprintf("/api/v1/containers/%s/id", containerName)
	agentAPI := fmt.Sprintf("%s%s", agentURL, endpoint)
	resp, err := s.restyAdapter.R().
		SetHeader("Accept", "application/json").
		SetHeader("X-Agent-Key", s.agentKey).
		Get(agentAPI)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		var bodyStr string
		if resp.Body() != nil {
			bodyStr = string(resp.Body())
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode(), bodyStr)
	}
	var result GetContainerIDByNameResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	return &result, nil

}

func (s *AgentService) IsContainerRunning(agentURL string, containerName string) (*IsContainerRunningResponse, error) {
	endpoint := fmt.Sprintf("/api/v1/containers/%s/running", containerName)
	agentAPI := fmt.Sprintf("%s%s", agentURL, endpoint)
	resp, err := s.restyAdapter.R().
		SetHeader("Accept", "application/json").
		SetHeader("X-Agent-Key", s.agentKey).
		Get(agentAPI)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		var bodyStr string
		if resp.Body() != nil {
			bodyStr = string(resp.Body())
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode(), bodyStr)
	}
	var result IsContainerRunningResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *AgentService) StartContainer(agentURL string, containerName string) (*StartContainerResponse, error) {
	resp, err := s.GetContainerIDByName(agentURL, containerName)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("/api/v1/containers/%s/start", resp.ID)
	agentAPI := fmt.Sprintf("%s%s", agentURL, endpoint)
	respF, err := s.restyAdapter.R().
		SetHeader("Accept", "application/json").
		SetHeader("X-Agent-Key", s.agentKey).
		Post(agentAPI)
	if err != nil {
		return nil, err
	}
	if respF.StatusCode() != 200 {
		var bodyStr string
		if respF.Body() != nil {
			bodyStr = string(respF.Body())
		}
		return nil, fmt.Errorf("request failed with status %d: %s", respF.StatusCode(), bodyStr)
	}
	var result StartContainerResponse
	if err := json.Unmarshal(respF.Body(), &result); err != nil {
		return nil, err
	}
	return &result, nil

}
func (s *AgentService) StopContainer(agentURL string, containerName string) (*StopContainerResponse, error) {
	resp, err := s.GetContainerIDByName(agentURL, containerName)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("/api/v1/containers/%s/stop", resp.ID)
	agentAPI := fmt.Sprintf("%s%s", agentURL, endpoint)
	respF, err := s.restyAdapter.R().
		SetHeader("Accept", "application/json").
		SetHeader("X-Agent-Key", s.agentKey).
		Post(agentAPI)
	if err != nil {
		return nil, err
	}
	if respF.StatusCode() != 200 {
		var bodyStr string
		if respF.Body() != nil {
			bodyStr = string(respF.Body())
		}
		return nil, fmt.Errorf("request failed with status %d: %s", respF.StatusCode(), bodyStr)
	}
	var result StopContainerResponse
	if err := json.Unmarshal(respF.Body(), &result); err != nil {
		return nil, err
	}
	return &result, nil

}
func (s *AgentService) RestartContainer(agentURL string, containerName string) (*RestartContainerResponse, error) {
	resp, err := s.GetContainerIDByName(agentURL, containerName)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("/api/v1/containers/%s/restart", resp.ID)
	agentAPI := fmt.Sprintf("%s%s", agentURL, endpoint)
	respF, err := s.restyAdapter.R().
		SetHeader("Accept", "application/json").
		SetHeader("X-Agent-Key", s.agentKey).
		Post(agentAPI)
	if err != nil || respF.StatusCode() != 200 {
		return nil, err
	}
	var result RestartContainerResponse
	if err := json.Unmarshal(respF.Body(), &result); err != nil {
		return nil, err
	}
	return &result, nil

}
func (s *AgentService) RemoveContainer(agentURL string, containerName string) (*RemoveContainerResponse, error) {
	resp, err := s.GetContainerIDByName(agentURL, containerName)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("/api/v1/containers/%s", resp.ID)
	agentAPI := fmt.Sprintf("%s%s", agentURL, endpoint)
	respF, err := s.restyAdapter.R().
		SetHeader("Accept", "application/json").
		SetHeader("X-Agent-Key", s.agentKey).
		SetQueryParam("force", "true").
		Delete(agentAPI)
	if err != nil || respF.StatusCode() != 200 {
		return nil, err
	}
	var result RemoveContainerResponse
	if err := json.Unmarshal(respF.Body(), &result); err != nil {
		return nil, err
	}
	return &result, nil

}

func (s *AgentService) GetContainerDefaults(agentURL string) (*GetContainerDefaultsResponse, error) {

	endpoint := "/api/v1/containers/defaults"
	agentAPI := fmt.Sprintf("%s%s", agentURL, endpoint)
	resp, err := s.restyAdapter.R().
		SetHeader("Accept", "application/json").
		SetHeader("X-Agent-Key", s.agentKey).
		Get(agentAPI)
	if err != nil || resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, err
	}

	var result GetContainerDefaultsResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	return &result, nil

}

func (s *AgentService) CreateContainer(agentURL string, req map[string]any) (*CreateContainerResponse, error) {

	endpoint := "/api/v1/containers"
	agentAPI := fmt.Sprintf("%s%s", agentURL, endpoint)
	s.log.Info().Msgf("%s", agentAPI)
	resp, err := s.restyAdapter.R().
		SetHeader("Accept", "application/json").
		SetHeader("X-Agent-Key", s.agentKey).
		SetBody(req).
		Post(agentAPI)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		var bodyStr string
		if resp.Body() != nil {
			bodyStr = string(resp.Body())
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode(), bodyStr)
	}

	var result CreateContainerResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	return &result, nil

}

func (s *AgentService) FetchMetrics(agentURL string) (*FetchMetricsResponse, error) {

	endpoint := "/api/v1/metrics"
	agentAPI := fmt.Sprintf("%s%s", agentURL, endpoint)
	s.log.Info().Msgf("%s", agentAPI)
	resp, err := s.restyAdapter.R().
		SetHeader("Accept", "application/json").
		SetHeader("X-Agent-Key", s.agentKey).
		Get(agentAPI)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		var bodyStr string
		if resp.Body() != nil {
			bodyStr = string(resp.Body())
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode(), bodyStr)
	}

	var result FetchMetricsResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	return &result, nil

}

// Select Best Agent for Container Schedule
func (s *AgentService) AgentLBSelector() (string, error) {
	ctx := context.Background()
	agentsData, err := s.RetrieveAllAgentData(ctx)
	if err != nil {
		return "", err
	}

	if len(agentsData) == 0 {
		return "", errors.New("no agents found")
	}

	type agentMetric struct {
		URL   string
		Score float64
		Err   error
	}

	ch := make(chan agentMetric, len(agentsData))

	for _, agent := range agentsData {
		go func(agent AgentServiceInfo) {
			proto := agent.MainHostProto
			if proto == "" {
				proto = "http"
			}
			url := fmt.Sprintf("%s://%s", proto, agent.MainHost)

			metrics, err := s.FetchMetrics(url)
			if err != nil {
				ch <- agentMetric{URL: url, Score: 0, Err: err}
				return
			}

			score := metrics.CPU + metrics.RAM
			ch <- agentMetric{URL: url, Score: score, Err: nil}
		}(agent)
	}

	var selectedURL string
	minScore := 201.0

	for i := 0; i < len(agentsData); i++ {
		am := <-ch
		if am.Err != nil {
			s.log.Error().Err(am.Err).Msgf("failed to fetch metrics for agent %s", am.URL)
			continue
		}
		if am.Score < minScore {
			minScore = am.Score
			selectedURL = am.URL
		}
	}

	if selectedURL == "" {
		return "", errors.New("failed to select any agent")
	}

	return selectedURL, nil
}

func (s *AgentService) RetrieveAllAgentData(ctx context.Context) ([]AgentServiceInfo, error) {
	pattern := "service:container_service:*"

	keys, err := s.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch keys: %w", err)
	}

	if len(keys) == 0 {
		return nil, nil
	}

	vals, err := s.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to MGet values: %w", err)
	}

	var agentServices []AgentServiceInfo
	for i, v := range vals {
		if vStr, ok := v.(string); ok {
			var c AgentServiceInfo
			if err := json.Unmarshal([]byte(vStr), &c); err != nil {
				s.log.Error().Err(err).Msgf("failed to unmarshal value for key %s", keys[i])
				continue
			}
			agentServices = append(agentServices, c)
		}
	}

	return agentServices, nil
}
