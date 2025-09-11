package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type ContainerInfo struct {
	User          string `json:"user"`
	ContainerName string `json:"container_name"`
	AgentHost     string `json:"agent_host"`
	CreatedAt     string `json:"created_at"`
}

type ContainerRegistryService struct {
	rdb *redis.Client
	log zerolog.Logger
}

func NewContainerRegistryService(rdb *redis.Client, log zerolog.Logger) *ContainerRegistryService {
	return &ContainerRegistryService{rdb, log}
}

func (s *ContainerRegistryService) Add(ctx context.Context, containerInfo *ContainerInfo) error {
	if containerInfo.CreatedAt == "" {
		containerInfo.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	data, err := json.Marshal(containerInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal container info: %w", err)
	}

	containerKey := fmt.Sprintf("container:%s", containerInfo.User)
	if err := s.rdb.Set(ctx, containerKey, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to save container info to Redis: %w", err)
	}

	s.log.Info().Msgf("Saved container info for %s -> %s -> %s", containerInfo.ContainerName, containerInfo.AgentHost, containerInfo.User)
	return nil
}

// Get container agent info
func (s *ContainerRegistryService) Get(ctx context.Context, user string) (*ContainerInfo, error) {
	containerKey := fmt.Sprintf("container:%s", user)
	val, err := s.rdb.Get(ctx, containerKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("container not found")
		}
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}

	var containerInfo ContainerInfo
	if err := json.Unmarshal([]byte(val), &containerInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal container info: %w", err)
	}

	return &containerInfo, nil
}

// Remove container agent info
func (s *ContainerRegistryService) Remove(ctx context.Context, user string) error {
	containerKey := fmt.Sprintf("container:%s", user)
	n, err := s.rdb.Del(ctx, containerKey).Result()
	if err != nil {
		return fmt.Errorf("failed to remove container info: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("container not found")
	}

	s.log.Info().Msgf("Removed container info for %s", user)
	return nil
}

// Get All Containers
func (s *ContainerRegistryService) GetAll(ctx context.Context) ([]ContainerInfo, error) {
	containerKeys, err := s.rdb.Keys(ctx, "container:*").Result()
	if err != nil {
		return nil, err
	}
	containers := []ContainerInfo{}
	for _, key := range containerKeys {
		val, err := s.rdb.Get(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		var c ContainerInfo
		if err := json.Unmarshal([]byte(val), &c); err == nil {
			containers = append(containers, c)
		}
	}
	return containers, nil

}
