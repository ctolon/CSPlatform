package xdiscovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type ServiceInstance struct {
	MainHost      string            `json:"mainHost"`
	MainHostProto string            `json:"mainHostProto"`
	HostPort      string            `json:"hostPort"`
	HostPortProto string            `mapstructure:"hostPortProto"`
	Version       string            `json:"version,omitempty"`
	Region        string            `json:"region,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
}

type Registry struct {
	rdb *redis.Client
	ttl time.Duration
	log zerolog.Logger
}

func NewRegistry(rdb *redis.Client, ttl time.Duration, log zerolog.Logger) *Registry {
	return &Registry{rdb: rdb, ttl: ttl, log: log}
}

func (r *Registry) instanceKey(serviceName, instanceID string) string {
	return "service:" + serviceName + ":" + instanceID
}

func (r *Registry) publishEvent(ctx context.Context, eventType, serviceName, instanceID string) {
	msg := map[string]string{
		"type":       eventType,
		"service":    serviceName,
		"instanceID": instanceID,
	}
	data, _ := json.Marshal(msg)
	r.rdb.Publish(ctx, "service-events", data)
}

func (r *Registry) Register(ctx context.Context, instanceID, serviceName string, inst ServiceInstance, realIP string) error {
	key := r.instanceKey(serviceName, instanceID)
	exists, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists > 0 {
		return fmt.Errorf("service instance already registered: %s:%s", serviceName, instanceID)
	}
	if inst.Tags != nil {
		inst.Tags["realIP"] = realIP
	} else {
		tags := make(map[string]string)
		inst.Tags = tags
		inst.Tags["realIP"] = realIP
	}
	data, _ := json.Marshal(inst)
	if err := r.rdb.Set(ctx, key, data, r.ttl).Err(); err != nil {
		return err
	}
	r.publishEvent(ctx, "register", serviceName, instanceID)
	return nil
}

func (r *Registry) Deregister(ctx context.Context, instanceID, serviceName string) error {
	key := r.instanceKey(serviceName, instanceID)
	n, err := r.rdb.Del(ctx, key).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("service not found")
	}
	r.publishEvent(ctx, "deregister", serviceName, instanceID)
	return nil
}

func (r *Registry) HealthCheck(ctx context.Context, instanceID, serviceName string) error {
	key := r.instanceKey(serviceName, instanceID)
	ttl, err := r.rdb.TTL(ctx, key).Result()
	if err != nil {
		return err
	}
	if ttl < 0 {
		r.log.Warn().Msgf("service instance not registered or expired: %s", instanceID)
		return errors.New("service instance not registered or expired")
	}
	return r.rdb.Expire(ctx, key, r.ttl).Err()
}

func (r *Registry) Discover(ctx context.Context, serviceName string) ([]ServiceInstance, error) {
	pattern := "service:" + serviceName + ":*"
	keys, err := r.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, errors.New("service not found")
	}

	vals, err := r.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	res := make([]ServiceInstance, 0, len(vals))
	for _, v := range vals {
		var inst ServiceInstance
		if vStr, ok := v.(string); ok {
			_ = json.Unmarshal([]byte(vStr), &inst)
			res = append(res, inst)
		}
	}
	return res, nil
}

func (r *Registry) GetActiveAgents(ctx context.Context, serviceName string) ([]ServiceInstance, error) {
	var agents []ServiceInstance
	pattern := fmt.Sprintf("agent:%s:*", serviceName)
	var cursor uint64
	for {
		keys, nextCursor, err := r.rdb.Scan(ctx, cursor, pattern, 100).Result() // 100 key batch
		if err != nil {
			return nil, fmt.Errorf("error scanning agent keys: %w", err)
		}

		for _, key := range keys {
			val, err := r.rdb.Get(ctx, key).Result()
			if err != nil {
				r.log.Error().Err(err).Msgf("failed to get value for key %s", key)
				continue
			}

			var inst ServiceInstance
			if err := json.Unmarshal([]byte(val), &inst); err != nil {
				r.log.Error().Err(err).Msgf("failed to unmarshal value for key %s", key)
				continue
			}

			agents = append(agents, inst)
		}

		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}

	if len(agents) == 0 {
		return nil, errors.New("no active agents found")
	}
	return agents, nil
}

/*
func (r *Registry) StartHeartbeat(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				keys, err := r.rdb.Keys(ctx, "service:*:*").Result()
				if err != nil {
					r.log.Printf("error fetching services for heartbeat: %v", err)
					continue
				}
				for _, key := range keys {
					ttl, err := r.rdb.TTL(ctx, key).Result()
					if err != nil || ttl < 0 {
						val, err := r.rdb.Get(ctx, key).Result()
						if err != nil {
							continue
						}
						var inst ServiceInstance
						_ = json.Unmarshal([]byte(val), &inst)
						// key format: service:<serviceName>:<instanceID>
						parts := strings.Split(key, ":")
						if len(parts) != 3 {
							continue
						}
						serviceName := parts[1]
						instanceID := parts[2]
						_ = r.Register(ctx, instanceID, serviceName, inst)
						r.log.Printf("re-registered instance %s:%s", serviceName, instanceID)
					}
				}

			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}
*/
