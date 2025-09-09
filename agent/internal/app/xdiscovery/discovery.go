package xdiscovery

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"a0/internal/app/service"
	"a0/internal/config"
)

type ServiceInstance struct {
	MainHost      string         `json:"mainHost"`
	MainHostProto string         `json:"mainHostProto"`
	HostPort      string         `json:"hostPort"`
	HostPortProto string         `json:"hostPortProto"`
	Version       string         `json:"version,omitempty"`
	Region        string         `json:"region,omitempty"`
	Tags          map[string]any `json:"tags,omitempty"`
}

type Agent struct {
	AgentKey    string
	Service     *service.DiscoveryService
	ServerURL   string
	Instance    ServiceInstance
	InstanceID  string
	ServiceName string
	Interval    time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	log         zerolog.Logger
}

func NewAgent(
	config *config.Config,
	service *service.DiscoveryService,
	interval time.Duration,
	log zerolog.Logger,
) *Agent {
	ctx, cancel := context.WithCancel(context.Background())
	serviceInstance := &ServiceInstance{
		MainHost:      config.AgentMetadata.MainHost,
		MainHostProto: config.AgentMetadata.MainHostProto,
		HostPort:      config.AgentMetadata.HostPort,
		HostPortProto: config.AgentMetadata.HostPortProto,
		Version:       config.AgentMetadata.Version,
		Region:        config.AgentMetadata.Region,
		Tags:          config.AgentMetadata.Tags,
	}
	return &Agent{
		AgentKey:    config.AgentMetadata.AgentKey,
		Service:     service,
		ServerURL:   config.AgentMetadata.ServerURL,
		ServiceName: config.AgentMetadata.ServiceName,
		InstanceID:  config.AgentMetadata.InstanceID,
		Instance:    *serviceInstance,
		Interval:    interval,
		ctx:         ctx,
		cancel:      cancel,
		log:         log,
	}
}

// Register agent
func (a *Agent) Register() error {
	req := &service.RegisterRequest{
		InstanceID:    a.InstanceID,
		ServiceName:   a.ServiceName,
		MainHost:      a.Instance.MainHost,
		MainHostProto: a.Instance.MainHostProto,
		HostPort:      a.Instance.HostPort,
		HostPortProto: a.Instance.HostPortProto,
		Version:       a.Instance.Version,
		Region:        a.Instance.Region,
		Tags:          a.Instance.Tags,
	}
	a.log.Info().Msg("Registering agent..")
	_, err := a.Service.Register(req)
	if err != nil {
		return err
	}
	log.Printf("Registered: %s - %s", a.ServiceName, a.InstanceID)
	return nil
}

// Deregister agent
func (a *Agent) Deregister() error {
	req := &service.DeregisterRequest{
		InstanceID:  a.InstanceID,
		ServiceName: a.ServiceName,
	}
	_, err := a.Service.Deregister(req)
	if err != nil {
		return err
	}
	log.Printf("Deregistered: %s - %s", a.ServiceName, a.InstanceID)
	return nil
}

// Start Hearbeat
func (a *Agent) StartHeartbeat() {
	ticker := time.NewTicker(a.Interval)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: tr,
	}
	go func() {
		for {
			select {
			case <-ticker.C:
				body := map[string]interface{}{
					"instanceID":  a.InstanceID,
					"serviceName": a.ServiceName,
				}
				data, _ := json.Marshal(body)
				req, err := http.NewRequest("POST", a.ServerURL+"/discovery/healthcheck", bytes.NewReader(data))
				if err != nil {
					log.Printf("Failed to create request: %v", err)
					continue
				}
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Agent-Key", a.AgentKey)
				resp, err := client.Do(req)
				if err != nil {
					log.Printf("Healthcheck failed, re-registering: %v", err)
					_ = a.Register()
				} else if resp.StatusCode != 200 {
					log.Printf("Healthcheck failed, re-registering: %v", resp.StatusCode)
					_ = a.Register()
				} else {
					resp.Body.Close()
					log.Printf("Healthcheck success: %s", a.InstanceID)
				}
			case <-a.ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// Agent start
func (a *Agent) Start() {
	interval := 15 * time.Second

	for {
		if err := a.Register(); err != nil {
			a.log.Error().Err(err).Msg("Register failed, will retry...")
			time.Sleep(interval)
			continue
		}

		a.log.Info().Msg("Register succeeded")
		break
	}

	a.StartHeartbeat()
}

func (a *Agent) Cancel() {
	if a.cancel != nil {
		a.cancel()
	}
}
