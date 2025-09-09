package service

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"

	"a0/internal/app/adapters"
	"a0/internal/config"
)

type RegisterResponse struct {
	Status string `json:"status"`
}

type DeregisterResponse struct {
	Status string `json:"status"`
}

type HealthcheckResponse struct {
	Status string `json:"status"`
}

type RegisterRequest struct {
	InstanceID    string         `json:"instanceID"`
	ServiceName   string         `json:"serviceName"`
	MainHost      string         `json:"mainHost"`
	MainHostProto string         `json:"mainHostProto"`
	HostPort      string         `json:"hostPort"`
	HostPortProto string         `json:"hostPortProto"`
	Version       string         `json:"version,omitempty"`
	Region        string         `json:"region,omitempty"`
	Tags          map[string]any `json:"tags,omitempty"`
}

type DeregisterRequest struct {
	InstanceID  string `json:"instanceID"`
	ServiceName string `json:"serviceName"`
}

type HealthcheckRequest struct {
	InstanceID  string `json:"instanceID"`
	ServiceName string `json:"serviceName"`
}

type DiscoveryService struct {
	restyClient *adapters.RestyClientAdapter
	config      *config.Config
	log         zerolog.Logger
}

func NewDiscoveryService(client *adapters.RestyClientAdapter, config *config.Config, log zerolog.Logger) *DiscoveryService {
	return &DiscoveryService{client, config, log}
}

func (s *DiscoveryService) Register(req *RegisterRequest) (*RegisterResponse, error) {
	endpoint := "/discovery/register"
	agentAPI := fmt.Sprintf("%s%s", s.config.AgentMetadata.ServerURL, endpoint)
	s.log.Info().Msgf("Register Request to: %s", agentAPI)
	resp, err := s.restyClient.R().
		SetHeader("Accept", "application/json").
		SetHeader("X-Agent-Key", s.config.AgentMetadata.AgentKey).
		SetBody(req).
		Post(agentAPI)
	s.log.Info().Msgf("Register status code: %d", resp.StatusCode())
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
	var result RegisterResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *DiscoveryService) Deregister(req *DeregisterRequest) (*DeregisterResponse, error) {
	endpoint := "/discovery/deregister"
	agentAPI := fmt.Sprintf("%s%s", s.config.AgentMetadata.ServerURL, endpoint)
	resp, err := s.restyClient.R().
		SetHeader("Accept", "application/json").
		SetHeader("X-Agent-Key", s.config.AgentMetadata.AgentKey).
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
	var result DeregisterResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	return &result, nil

}

func (s *DiscoveryService) Healtcheck(req *HealthcheckRequest) (*HealthcheckResponse, error) {
	endpoint := "/discovery/healthcheck"
	agentAPI := fmt.Sprintf("%s%s", s.config.AgentMetadata.ServerURL, endpoint)
	resp, err := s.restyClient.R().
		SetHeader("Accept", "application/json").
		SetHeader("X-Agent-Key", s.config.AgentMetadata.AgentKey).
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
	var result HealthcheckResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	return &result, nil

}
