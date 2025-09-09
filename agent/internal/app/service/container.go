package service

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/rs/zerolog"

	"a0/internal/config"
)

type CreateContainerRequest struct {
	Image      string            `json:"image"`
	Name       string            `json:"name,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	Volumes    []string          `json:"volumes,omitempty"`
	Expose     []string          `json:"expose,omitempty"`
	Ports      []string          `json:"ports,omitempty"`
	CPUQuota   int64             `json:"cpuQuota,omitempty"`
	Memory     string            `json:"memory,omitempty"`
	Sysctls    map[string]string `json:"sysctls,omitempty"`
	Network    string            `json:"network,omitempty"`
	Restart    string            `json:"restart,omitempty"`
	ExtraHosts []string          `json:"extra_hosts,omitempty"`
}

type ConfigDefaultsResponse struct {
	Image      string            `json:"image"`
	Name       string            `json:"name,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	Volumes    []string          `json:"volumes,omitempty"`
	Expose     []string          `json:"expose,omitempty"`
	Ports      []string          `json:"ports,omitempty"`
	CPUQuota   int64             `json:"cpuQuota,omitempty"`
	Memory     string            `json:"memory,omitempty"`
	Sysctls    map[string]string `json:"sysctls,omitempty"`
	Network    string            `json:"network,omitempty"`
	Restart    string            `json:"restart,omitempty"`
	ExtraHosts []string          `json:"extra_hosts,omitempty"`
}

type ContainerService struct {
	cli    *client.Client
	config *config.Config
	log    zerolog.Logger
}

func NewContainerService(cli *client.Client, config *config.Config, log zerolog.Logger) *ContainerService {
	return &ContainerService{cli, config, log}
}

func RemoveDuplicateEnv(envs []string) []string {
	envMap := make(map[string]string)
	order := []string{}

	for _, env := range envs {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]

		if _, exists := envMap[key]; !exists {
			order = append(order, key)
		}
		envMap[key] = value
	}

	result := []string{}
	for _, key := range order {
		result = append(result, fmt.Sprintf("%s=%s", key, envMap[key]))
	}
	return result
}

func removeDuplicates(input []string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, v := range input {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// normalize env: uppercase key
func normalizeEnv(env map[string]any) []string {
	result := []string{}
	for k, v := range env {
		key := strings.ToUpper(k)
		result = append(result, fmt.Sprintf("%s=%v", key, v))
	}
	return result
}

func flattenSysctls(input map[string]any) map[string]string {
	result := map[string]string{}

	var walk func(prefix string, val any)
	walk = func(prefix string, val any) {
		switch v := val.(type) {
		case map[string]any:
			for k, sub := range v {
				newKey := k
				if prefix != "" {
					newKey = prefix + "." + k
				}
				walk(newKey, sub)
			}
		case string:
			result[prefix] = v
		case int, int32, int64, float32, float64, bool:
			result[prefix] = fmt.Sprintf("%v", v)
		default:
			fmt.Printf("Warning: unsupported sysctl type %T for key %s\n", v, prefix)
		}
	}

	for k, v := range input {
		walk(k, v)
	}

	return result
}

func mergeStringSlice(defaults, overrides []string) []string {
	m := map[string]struct{}{}
	for _, v := range defaults {
		m[v] = struct{}{}
	}
	for _, v := range overrides {
		m[v] = struct{}{}
	}
	res := []string{}
	for k := range m {
		res = append(res, k)
	}
	return res
}

func formatMemory(bytes int64) string {
	const (
		GB int64 = 1024 * 1024 * 1024
		MB int64 = 1024 * 1024
		KB int64 = 1024
	)

	if bytes <= 0 {
		return ""
	}

	if bytes%GB == 0 {
		return fmt.Sprintf("%dg", bytes/GB)
	}
	if bytes%MB == 0 {
		return fmt.Sprintf("%dm", bytes/MB)
	}
	if bytes%KB == 0 {
		return fmt.Sprintf("%dk", bytes/KB)
	}
	return fmt.Sprintf("%d", bytes)
}

func (s *ContainerService) buildExposedPorts(expose []int) nat.PortSet {
	ports := nat.PortSet{}
	for _, p := range expose {
		portKey := nat.Port(fmt.Sprintf("%d/tcp", p))
		ports[portKey] = struct{}{}
	}
	return ports
}

func (s *ContainerService) buildPortBindingsWithExpose(ports []string) (nat.PortMap, nat.PortSet, error) {
	portBindings := nat.PortMap{}
	exposePorts := nat.PortSet{}

	for _, p := range ports {
		parts := strings.Split(p, ":")
		if len(parts) != 2 {
			return nil, nil, fmt.Errorf("invalid port mapping: %s", p)
		}
		hostPort := parts[0]
		containerPort := parts[1]

		portKey := nat.Port(fmt.Sprintf("%s/tcp", containerPort))
		exposePorts[portKey] = struct{}{}
		portBindings[portKey] = []nat.PortBinding{{HostPort: hostPort}}
	}
	return portBindings, exposePorts, nil
}

func (s *ContainerService) buildPortBindings(ports []string) (nat.PortMap, error) {
	portBindings := nat.PortMap{}

	for _, p := range ports {
		parts := strings.Split(p, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid port mapping: %s", p)
		}
		hostPort := parts[0]
		containerPort := parts[1]

		portKey := nat.Port(fmt.Sprintf("%s/tcp", containerPort))
		portBindings[portKey] = []nat.PortBinding{{HostPort: hostPort}}
	}
	return portBindings, nil
}

func (s *ContainerService) buildEnv(envMap map[string]string) []string {
	envs := []string{}
	for k, v := range envMap {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	return envs
}

func (s *ContainerService) parseMemoryLimitWithSanityCheck(mem string) (int64, error) {
	// empty -> no limit
	if strings.TrimSpace(mem) == "" {
		return 0, nil
	}

	m := strings.TrimSpace(mem)
	m = strings.ToLower(m)

	// split numeric prefix and unit suffix
	numEnd := len(m)
	for i, ch := range m {
		if !(ch >= '0' && ch <= '9' || ch == '.') {
			numEnd = i
			break
		}
	}

	numStr := strings.TrimSpace(m[:numEnd])
	unitStr := strings.TrimSpace(m[numEnd:])

	if numStr == "" {
		return 0, fmt.Errorf("invalid memory limit: %q", mem)
	}

	// remove optional trailing 'b' (e.g. "mb", "gb")
	unitStr = strings.TrimSuffix(unitStr, "b")

	// parse numeric part as float to allow decimals like "1.5g"
	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory value: %w", err)
	}
	if val < 0 {
		return 0, fmt.Errorf("memory must be non-negative")
	}

	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	var multiplier float64
	switch unitStr {
	case "", "b":
		multiplier = 1
	case "k":
		multiplier = float64(KB)
	case "m":
		multiplier = float64(MB)
	case "g":
		multiplier = float64(GB)
	case "t":
		multiplier = float64(TB)
	default:
		return 0, fmt.Errorf("invalid memory unit: %q", unitStr)
	}

	bytesF := val * multiplier

	// overflow / sanity checks
	if bytesF < 0 || bytesF > float64(math.MaxInt64) {
		return 0, fmt.Errorf("memory limit out of range")
	}

	return int64(bytesF), nil
}

func (s *ContainerService) parseMemoryLimit(mem string) (int64, error) {
	if mem == "" {
		return 0, nil
	}

	mem = strings.TrimSpace(strings.ToLower(mem))
	var multiplier int64 = 1

	switch {
	case strings.HasSuffix(mem, "g"):
		multiplier = 1024 * 1024 * 1024
		mem = strings.TrimSuffix(mem, "g")
	case strings.HasSuffix(mem, "m"):
		multiplier = 1024 * 1024
		mem = strings.TrimSuffix(mem, "m")
	case strings.HasSuffix(mem, "k"):
		multiplier = 1024
		mem = strings.TrimSuffix(mem, "k")
	default:
	}

	value, err := strconv.ParseInt(mem, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory limit: %w", err)
	}

	return value * multiplier, nil
}

func (s *ContainerService) parseRestartPolicy(restart string) container.RestartPolicy {
	switch restart {
	case "always":
		return container.RestartPolicy{Name: "always"}
	case "on-failure":
		return container.RestartPolicy{Name: "on-failure"}
	case "unless-stopped":
		return container.RestartPolicy{Name: "unless-stopped"}
	case "no":
		fallthrough
	default:
		return container.RestartPolicy{Name: "no"}
	}
}

/********/

func (s *ContainerService) PullImage(cli *client.Client, ctx context.Context, imageName string) error {
	out, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return err
	}
	defer out.Close()
	io.Copy(os.Stdout, out)
	return nil
}

func (s *ContainerService) buildContainerConfig() *container.Config {
	containerConfig := &container.Config{}

	// expose
	if len(s.config.ContainerTemplate.Expose) > 0 {
		containerConfig.ExposedPorts = s.buildExposedPorts(s.config.ContainerTemplate.Expose)
	}

	// env
	if len(s.config.ContainerTemplate.Environment) > 0 {
		containerConfig.Env = RemoveDuplicateEnv(normalizeEnv(s.config.ContainerTemplate.Environment))
	}

	// image_name
	if s.config.ContainerTemplate.ImageName != "" {
		containerConfig.Image = s.config.ContainerTemplate.ImageName
	}

	return containerConfig
}

func (s *ContainerService) buildHostConfig() *container.HostConfig {
	hostConfig := &container.HostConfig{}

	// set sysctls
	if len(s.config.ContainerTemplate.Sysctls) > 0 {
		hostConfig.Sysctls = flattenSysctls(s.config.ContainerTemplate.Sysctls)
	}

	// set port binding
	if len(s.config.ContainerTemplate.Ports) > 0 {
		if bindings, err := s.buildPortBindings(s.config.ContainerTemplate.Ports); err == nil {
			hostConfig.PortBindings = bindings
		}
	}

	// set restart policy
	hostConfig.RestartPolicy = s.parseRestartPolicy(s.config.ContainerTemplate.Restart)

	// set extra hosts
	filteredHosts := []string{}
	for _, h := range s.config.ContainerTemplate.ExtraHost {
		h = strings.TrimSpace(h)
		if h != "" {
			filteredHosts = append(filteredHosts, h)
		}
	}
	hostConfig.ExtraHosts = filteredHosts

	// set cpus
	if s.config.ContainerTemplate.Cpus > 0 {
		hostConfig.Resources.NanoCPUs = int64(s.config.ContainerTemplate.Cpus) * 1_000_000_000
	} else {
		hostConfig.Resources.NanoCPUs = 1_000_000_000
	}

	// set mem_limit
	memBytes, err := s.parseMemoryLimit(s.config.ContainerTemplate.MemLimit)
	if err != nil || memBytes == 0 {
		memBytes = 1 * 1024 * 1024 * 1024
	}
	hostConfig.Resources.Memory = memBytes

	// set network
	for netName := range s.config.ContainerTemplate.Networks {
		if netName != "" {
			hostConfig.NetworkMode = container.NetworkMode(netName)
			break
		}
	}

	// set bind volumes
	filteredVolumes := []string{}
	for _, v := range s.config.ContainerTemplate.Volumes {
		v = strings.TrimSpace(v)
		if v != "" {
			filteredVolumes = append(filteredVolumes, v)
		}
	}
	hostConfig.Binds = filteredVolumes

	return hostConfig
}

func (s *ContainerService) CreateContainer(req *CreateContainerRequest) (*container.CreateResponse, error) {

	// DEFAULTS
	// ***************
	ctx := context.Background()
	containerName := s.config.ContainerTemplate.ContainerName
	defaultContainerConfig := s.buildContainerConfig()
	defaultHostConfig := s.buildHostConfig()

	// OVERRIDE
	// ***************

	// set image
	if req.Image != "" {
		defaultContainerConfig.Image = req.Image
	}

	// set container_name
	if req.Name != "" {
		containerName = req.Name
	}

	// set restart
	if req.Restart != "" {
		defaultHostConfig.RestartPolicy = s.parseRestartPolicy(req.Restart)
	}

	// set expose
	if len(req.Expose) > 0 {
		exposePorts := nat.PortSet{}
		for _, p := range req.Expose {
			exposePorts[nat.Port(fmt.Sprintf("%s/tcp", p))] = struct{}{}
		}
		defaultContainerConfig.ExposedPorts = exposePorts
	}

	// set ports
	if len(req.Ports) > 0 {
		portBinds, err := s.buildPortBindings(req.Ports)
		if err == nil {
			defaultHostConfig.PortBindings = portBinds
		}
	}

	// set sysctls
	if req.Sysctls != nil {
		defaultHostConfig.Sysctls = req.Sysctls
	}

	// set cpus
	if req.CPUQuota > 0 {
		defaultHostConfig.Resources.NanoCPUs = int64(req.CPUQuota) * 1_000_000_000
	} else if defaultHostConfig.Resources.NanoCPUs == 0 {
		defaultHostConfig.Resources.NanoCPUs = 1_000_000_000
	}

	// set mem_limit
	if req.Memory != "" {
		memBytes, err := s.parseMemoryLimit(req.Memory)
		if err != nil || memBytes == 0 {
			memBytes = 1 * 1024 * 1024 * 1024
		}
		defaultHostConfig.Resources.Memory = memBytes
	}

	// set extra_hosts
	if len(req.ExtraHosts) > 0 {
		filtered := []string{}
		for _, h := range req.ExtraHosts {
			h = strings.TrimSpace(h)
			if h != "" {
				filtered = append(filtered, h)
			}
		}
		defaultHostConfig.ExtraHosts = removeDuplicates(filtered)
	}
	// set volumes
	if len(req.Volumes) > 0 {
		filtered := []string{}
		for _, v := range req.Volumes {
			v = strings.TrimSpace(v)
			if v != "" {
				filtered = append(filtered, v)
			}
		}
		defaultHostConfig.Binds = removeDuplicates(append(defaultHostConfig.Binds, filtered...))
	}

	// set env
	if len(req.Env) > 0 {
		defaultContainerConfig.Env = RemoveDuplicateEnv(append(defaultContainerConfig.Env, s.buildEnv(req.Env)...))
	}

	// set network
	if req.Network != "" {
		defaultHostConfig.NetworkMode = container.NetworkMode(req.Network)
	}

	resp, err := s.cli.ContainerCreate(
		ctx,
		defaultContainerConfig,
		defaultHostConfig,
		nil,
		nil,
		containerName,
	)

	return &resp, err

}

// StopContainer
func (s *ContainerService) StopContainer(containerID string) error {
	ctx := context.Background()
	return s.cli.ContainerStop(ctx, containerID, container.StopOptions{})
}

// StartContainer
func (s *ContainerService) StartContainer(containerID string) error {
	ctx := context.Background()
	return s.cli.ContainerStart(ctx, containerID, container.StartOptions{})
}

// RestartContainer
func (s *ContainerService) RestartContainer(containerID string) error {
	ctx := context.Background()
	return s.cli.ContainerRestart(ctx, containerID, container.StopOptions{})
}

// RemoveContainer
func (s *ContainerService) RemoveContainer(containerID string, force bool) error {
	ctx := context.Background()
	return s.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: force,
	})
}

// ListContainers
func (s *ContainerService) ListContainers(all bool) ([]container.Summary, error) {
	ctx := context.Background()
	containers, err := s.cli.ContainerList(ctx, container.ListOptions{
		All: all,
	})
	if err != nil {
		return nil, err
	}
	return containers, nil
}

// LogsContainer
func (s *ContainerService) LogsContainer(containerID string, tail string) (string, error) {
	ctx := context.Background()
	out, err := s.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
	})
	if err != nil {
		return "", err
	}
	defer out.Close()

	logs, err := io.ReadAll(out)
	if err != nil {
		return "", err
	}
	return string(logs), nil
}

// InspectContainer
func (s *ContainerService) InspectContainer(containerID string) (container.InspectResponse, error) {
	ctx := context.Background()
	inspect, err := s.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return container.InspectResponse{}, err
	}
	return inspect, nil
}

func (s *ContainerService) ListCodeServerContainersPrefix() ([]container.Summary, error) {
	ctx := context.Background()

	args := filters.NewArgs()
	args.Add("name", "code-server")

	containers, err := s.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: args,
	})
	if err != nil {
		return nil, err
	}

	filtered := []container.Summary{}
	for _, c := range containers {
		for _, name := range c.Names {
			name = strings.TrimPrefix(name, "/")
			if strings.HasPrefix(name, "code-server-") {
				filtered = append(filtered, c)
				break
			}
		}
	}

	return filtered, nil
}

func (s *ContainerService) GetContainerIDByName(name string) (string, error) {
	ctx := context.Background()

	args := filters.NewArgs()
	args.Add("name", name)

	containers, err := s.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: args,
	})
	if err != nil {
		return "", err
	}

	if len(containers) == 0 {
		return "", fmt.Errorf("container with name '%s' not found", name)
	}

	return containers[0].ID, nil
}

func (s *ContainerService) GetConfigDefaults() (*ConfigDefaultsResponse, error) {
	containerName := s.config.ContainerTemplate.ContainerName
	defaultContainerConfig := s.buildContainerConfig()
	defaultHostConfig := s.buildHostConfig()

	// memory
	memBytes := defaultHostConfig.Resources.Memory
	if memBytes == 0 {
		// fallback default 1GB
		memBytes = 1 * 1024 * 1024 * 1024
	}
	memStr := formatMemory(memBytes)

	// cpus (int64)
	var cpus int64
	if s.config.ContainerTemplate.Cpus > 0 {
		cpus = int64(s.config.ContainerTemplate.Cpus) * 1_000_000_000
	} else {
		cpus = 1_000_000_000
	}

	// env string → map[string]string
	envMap := map[string]string{}
	for _, e := range defaultContainerConfig.Env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// expose map → []string
	exposeList := []string{}
	for k := range defaultContainerConfig.ExposedPorts {
		exposeList = append(exposeList, string(k))
	}

	// ports map → []string ("host:container")
	portList := []string{}
	for k, v := range defaultHostConfig.PortBindings {
		for _, binding := range v {
			portList = append(portList, fmt.Sprintf("%s:%s", binding.HostPort, k.Port()))
		}
	}

	resp := &ConfigDefaultsResponse{
		Image:      defaultContainerConfig.Image,
		Name:       containerName,
		Env:        envMap,
		Volumes:    defaultHostConfig.Binds,
		Expose:     exposeList,
		Ports:      portList,
		CPUQuota:   cpus,
		Memory:     memStr,
		Sysctls:    defaultHostConfig.Sysctls,
		Network:    defaultHostConfig.NetworkMode.NetworkName(),
		Restart:    string(defaultHostConfig.RestartPolicy.Name),
		ExtraHosts: defaultHostConfig.ExtraHosts,
	}

	return resp, nil
}

// IsContainerExist checks if a container with the given name exists (any state)
func (s *ContainerService) IsContainerExist(name string) (bool, error) {
	ctx := context.Background()
	args := filters.NewArgs()
	args.Add("name", name)

	containers, err := s.cli.ContainerList(ctx, container.ListOptions{
		All:     true, // all containers
		Filters: args,
	})
	if err != nil {
		return false, err
	}

	return len(containers) > 0, nil
}

// IsContainerRunning checks if a container with the given name is currently running
func (s *ContainerService) IsContainerRunning(name string) (bool, error) {
	ctx := context.Background()
	args := filters.NewArgs()
	args.Add("name", name)

	containers, err := s.cli.ContainerList(ctx, container.ListOptions{
		All:     false, // only running containers
		Filters: args,
	})
	if err != nil {
		return false, err
	}

	return len(containers) > 0, nil
}
