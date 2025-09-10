package handlers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"v0/internal/app/service"
	"v0/internal/config"
)

type UserInfo struct {
	Username string `json:"username"`
	UID      int    `json:"uid"`
	GID      int    `json:"gid"`
}

type ContainerFormData struct {
	Agent               string
	Image               string
	Name                string
	Memory              string
	CPUQuota            string
	Restart             string
	Network             string
	AgentOptions        []string
	MemoryOptions       []string
	CPUOptions          []string
	AllowEditImage      bool
	AllowEditName       bool
	AllowEditMemory     bool
	AllowEditCPU        bool
	AllowEditRestart    bool
	AllowEditNetwork    bool
	AllowEditPorts      bool
	AllowEditExpose     bool
	AllowEditVolumes    bool
	AllowEditExtraHosts bool
	AllowEditEnv        bool
	AllowEditSysctls    bool
	JSON                template.JS
}

type ContainerHandler struct {
	userInfoService *service.UserInfoService
	tmpl         *template.Template
	agentService *service.AgentService
	log          zerolog.Logger
	reg          *service.ContainerRegistryService
	config       *config.AppConfig
}

func NewContainerHandler(
	userInfoService *service.UserInfoService,
	tmpl *template.Template,
	agentService *service.AgentService,
	log zerolog.Logger,
	reg *service.ContainerRegistryService,
	config *config.AppConfig,
) *ContainerHandler {
	return &ContainerHandler{userInfoService, tmpl, agentService, log, reg, config}
}

func (h *ContainerHandler) ShowFormCreate(c echo.Context) error {

	// Check user have container or not
	if resp, err := h.reg.Get(context.Background(), c.Get("username").(string)); err == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   fmt.Sprintf("%s already have container", resp.User),
			"details": fmt.Sprintf("Agent Host: %s Container Name: %s -> Created At %s", resp.AgentHost, resp.ContainerName, resp.CreatedAt),
		})
	}

	// --- Agent LB Selector ---
	agentInfo, err := h.agentService.AgentLBSelector()
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to select agent: %v", err))
	}
	agentURL := agentInfo.URL

	// Get All Agents Options
	ctx := context.Background()
	var agentOptions []string
	if agents, err := h.agentService.RetrieveAllAgentData(ctx); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch agents: %v", err))
	} else {
		for _, data := range agents {
			agentHost := fmt.Sprintf("%s://%s", data.MainHostProto, data.MainHost)
			agentOptions = append(agentOptions, agentHost)
		}
	}
	agentOptions = append(agentOptions, "Auto")

	defaults, err := h.agentService.GetContainerDefaults(agentURL)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch defaults: %v", err))
	}

	// edit policies
	defaults.AllowEditCPU = h.config.ContainerAllowEditCPU
	defaults.AllowEditEnv = h.config.ContainerAllowEditEnv
	defaults.AllowEditExpose = h.config.ContainerAllowEditEnv
	defaults.AllowEditExtraHosts = h.config.ContainerAllowEditExtraHosts
	defaults.AllowEditImage = h.config.ContainerAllowEditImage
	defaults.AllowEditMemory = h.config.ContainerAllowEditMemory
	defaults.AllowEditName = h.config.ContainerAllowEditName
	defaults.AllowEditNetwork = h.config.ContainerAllowEditNetwork
	defaults.AllowEditPorts = h.config.ContainerAllowEditPorts
	defaults.AllowEditRestart = h.config.ContainerAllowEditRestart
	defaults.AllowEditSysctls = h.config.ContainerAllowEditSysctls
	defaults.AllowEditVolumes = h.config.ContainerAllowEditVolumes

	// add nfs volume
	if h.config.AppNFSHome != "" {
		defaults.Volumes = append(defaults.Volumes, fmt.Sprintf("%s/%s:/config", h.config.AppNFSHome, c.Get("username").(string)))
	}

	// marshall data
	jsonData, _ := json.Marshal(defaults)

	data := ContainerFormData{
		Agent:               "Auto",
		Image:               defaults.Image,
		Name:                fmt.Sprintf("code-server-%s", c.Get("username").(string)),
		Memory:              defaults.Memory,
		CPUQuota:            fmt.Sprintf("%d", defaults.CPUQuota/1_000_000_000), // nano cpus
		Restart:             defaults.Restart,
		Network:             defaults.Network,
		AgentOptions:        agentOptions,
		MemoryOptions:       []string{"2g", "4g", "8g", "16g", "32g"},
		CPUOptions:          []string{"2", "4", "8", "16", "32"},
		JSON:                template.JS(jsonData),
		AllowEditImage:      defaults.AllowEditImage,
		AllowEditName:       defaults.AllowEditName,
		AllowEditMemory:     defaults.AllowEditMemory,
		AllowEditCPU:        defaults.AllowEditCPU,
		AllowEditRestart:    defaults.AllowEditRestart,
		AllowEditNetwork:    defaults.AllowEditNetwork,
		AllowEditPorts:      defaults.AllowEditPorts,
		AllowEditExpose:     defaults.AllowEditExpose,
		AllowEditVolumes:    defaults.AllowEditVolumes,
		AllowEditExtraHosts: defaults.AllowEditExtraHosts,
		AllowEditEnv:        defaults.AllowEditEnv,
		AllowEditSysctls:    defaults.AllowEditSysctls,
	}

	return h.tmpl.ExecuteTemplate(c.Response(), "container-home.go.tmpl", data)
}

func (h *ContainerHandler) StopContainer(c echo.Context) error {
	ctx := context.Background()
	cntInfo, err := h.reg.Get(ctx, c.Get("username").(string))
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to stop container: %v", err))
	}

	_, err = h.agentService.StopContainer(cntInfo.AgentHost, cntInfo.ContainerName)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to stop container: %v", err))
	}

	return c.Redirect(302, "/csplatform/home")
}

func (h *ContainerHandler) RestartContainer(c echo.Context) error {
	ctx := context.Background()
	cntInfo, err := h.reg.Get(ctx, c.Get("username").(string))
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to stop container: %v", err))
	}
	_, err = h.agentService.RestartContainer(cntInfo.AgentHost, cntInfo.ContainerName)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to restart container: %v", err))
	}
	return c.Redirect(302, "/csplatform/home")
}

func (h *ContainerHandler) StartContainer(c echo.Context) error {
	ctx := context.Background()
	cntInfo, err := h.reg.Get(ctx, c.Get("username").(string))
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to stop container: %v", err))
	}
	_, err = h.agentService.StartContainer(cntInfo.AgentHost, cntInfo.ContainerName)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to stop container: %v", err))
	}
	return c.Redirect(302, "/csplatform/home")
}

func (h *ContainerHandler) RemoveContainer(c echo.Context) error {
	ctx := context.Background()
	cntInfo, err := h.reg.Get(ctx, c.Get("username").(string))
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to stop container: %v", err))
	}
	_, err = h.agentService.RemoveContainer(cntInfo.AgentHost, cntInfo.ContainerName)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to remove container: %v", err))
	}
	h.reg.Remove(ctx, c.Get("username").(string))
	return c.Redirect(302, "/csplatform/home")
}

func (h *ContainerHandler) CreateContainerRequest(c echo.Context) error {

	if err := c.Request().ParseForm(); err != nil {
		return c.String(http.StatusBadRequest, "Failed to parse form")
	}

	agentForm := c.FormValue("agent")
	name := c.FormValue("name")

	ctx := context.Background()
	if _, err := h.reg.Get(ctx, c.Get("username").(string)); err == nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("User already have container: %s", name))
	} else if err.Error() != "container not found" {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to get registry: %v", err))
	}

	image := c.FormValue("image")
	memory := c.FormValue("memory")
	cpuQuotaStr := c.FormValue("cpuQuota")
	cpuQuota, err := strconv.ParseInt(cpuQuotaStr, 10, 64) // convert to int64
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid cpuQuota value",
		})
	}
	restart := c.FormValue("restart")
	network := c.FormValue("network")

	ports := c.Request().Form["ports[]"]           // []string
	expose := c.Request().Form["expose[]"]         // []string
	volumes := c.Request().Form["volumes[]"]       // []string
	extraHosts := c.Request().Form["extraHosts[]"] // []string

	envKeys := c.Request().Form["env_key[]"]
	envVals := c.Request().Form["env_val[]"]
	sysctlsKeys := c.Request().Form["sysctls_key[]"]
	sysctlsVals := c.Request().Form["sysctls_val[]"]

	env := map[string]string{}
	for i := range envKeys {
		val := ""
		if i < len(envVals) {
			val = envVals[i]
		}
		env[envKeys[i]] = val
	}

	// Auto PUID PGID Fetch from PAM API
	if h.config.PAMAPIUrl != "" {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		client := &http.Client{
			Timeout: 30 * time.Second,
			Transport: tr,
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		url := fmt.Sprintf("%s/%s", h.config.PAMAPIUrl, c.Get("username").(string))
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return c.JSON(500, echo.Map{
				"Error": "Fatal error on Pam Request CODE 001",
			})
		}

		req.Header.Set("X-Api-Key", h.config.PAMAuthAPIKey)

		resp, err := client.Do(req)
		if err != nil {
			return c.JSON(500, echo.Map{
				"Error": "Fatal error on Pam Request CODE 002",
			})
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return c.JSON(http.StatusUnprocessableEntity, echo.Map{
				"Error": string(body),
			})
		}
		var info UserInfo
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return c.JSON(http.StatusUnprocessableEntity, echo.Map{
				"Error": err.Error(),
			})
		}
		env["PUID"] = strconv.Itoa(info.UID)
		env["PGID"] = strconv.Itoa(info.GID)
	}

	if userInfo, err := h.userInfoService.GetUserGroupInfo(c.Get("username").(string)); err == nil{
		env["PUID"] = userInfo.UID
		env["PGID"] = userInfo.GID
	}

	if env["PUID"] == "" {
		return c.JSON(http.StatusUnprocessableEntity, echo.Map{
			"Error": "PUID env variable is required",
		})
	}
	if env["PGID"] == "" {
		return c.JSON(http.StatusUnprocessableEntity, echo.Map{
			"Error": "PGID env variable is required",
		})
	}

	sysctls := map[string]string{}
	for i := range sysctlsKeys {
		val := ""
		if i < len(sysctlsVals) {
			val = sysctlsVals[i]
		}
		sysctls[sysctlsKeys[i]] = val
	}

	containerData := map[string]interface{}{
		"image":      image,
		"name":       name,
		"memory":     memory,
		"cpuQuota":   cpuQuota,
		"restart":    restart,
		"network":    network,
		"ports":      ports,
		"expose":     expose,
		"volumes":    volumes,
		"extraHosts": extraHosts,
		"env":        env,
		"sysctls":    sysctls,
	}

	for k, v := range containerData {
		switch val := v.(type) {
		case string:
			if val == "" {
				delete(containerData, k)
			}
		case []string:
			if len(val) == 0 {
				delete(containerData, k)
			}
		case map[string]string:
			if len(val) == 0 {
				delete(containerData, k)
			}
		}
	}

	// --- Agent LB Selector ---
	var agentURL string
	if strings.ToLower(agentForm) == "auto" {
		agentInfo, err := h.agentService.AgentLBSelector()
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to select agent with LB: %v", err))
		}
		agentURL = agentInfo.URL
		// spark driver host assign
		if agentInfo.Info.Tags != nil {
			if sparkDriverHost, ok := agentInfo.Info.Tags["spark_driver_host"]; ok {
				appendSparkDriverHost(env, sparkDriverHost, "SPARK_SUBMIT_OPTS", "SPARK3_SUBMIT_OPTS")
				appendSparkDriverBindAddress(env, "SPARK_SUBMIT_OPTS", "SPARK3_SUBMIT_OPTS")
			}
		}
		// --- Manual Selector ---
	} else {
		agentURL = agentForm
		agentTags, err := h.agentService.GetAgentTags(agentForm)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to create container: %v", err))
		}
		// spark driver host assign
		if sparkDriverHost, ok := agentTags["spark_driver_host"].(string); ok {
			appendSparkDriverHost(env, sparkDriverHost, "SPARK_SUBMIT_OPTS", "SPARK3_SUBMIT_OPTS")
			appendSparkDriverBindAddress(env, "SPARK_SUBMIT_OPTS", "SPARK3_SUBMIT_OPTS")
		}
	}

	// Create container with API request on agent
	_, err = h.agentService.CreateContainer(agentURL, containerData)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to create container: %v", err))
	}

	// Save Container info to redis
	containerInfo := &service.ContainerInfo{
		User:          c.Get("username").(string),
		ContainerName: name,
		AgentHost:     agentURL,
	}
	if err := h.reg.Add(ctx, containerInfo); err != nil {
		h.log.Error().Err(err).Msgf("failed to save container-agent info for %s", name)
		// Attempt to delete the container on the agent
		if delResp, delErr := h.agentService.RemoveContainer(agentURL, name); delErr != nil {
			h.log.Error().Err(delErr).Msgf("failed to rollback container %s on agent %s", name, agentURL)
		} else {
			h.log.Info().Msgf("Rollback success: %v", delResp)
		}
	}

	return c.Redirect(302, "/csplatform/home")

}

func appendSparkDriverHost(env map[string]string, sparkDriverHost string, keys ...string) {
    for _, key := range keys {
        prev := env[key]
        env[key] = strings.TrimSpace(fmt.Sprintf("-Dspark.driver.host=%s %s", sparkDriverHost, prev))
    }
}

func appendSparkDriverBindAddress(env map[string]string, keys ...string) {
    for _, key := range keys {
        prev := env[key]
        env[key] = strings.TrimSpace(fmt.Sprintf("-Dspark.driver.bindAddress=0.0.0.0 %s", prev))
    }
}