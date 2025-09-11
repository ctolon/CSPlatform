package routes

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"a0/internal/app/adapters"
	"a0/internal/app/api/handlers"
	"a0/internal/app/inmemory"
	"a0/internal/app/security"
	"a0/internal/app/service"
	"a0/internal/app/utils"
	"a0/internal/app/xdiscovery"
	"a0/internal/app/xsession"
	"a0/internal/config"
)

func JWTAuthMiddleware(
	authService *service.AuthService,
	errorHeader string,
	withTLS bool,
	log zerolog.Logger,
	requireGroups []string,
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			requestPath := c.Request().URL.Path

			if strings.HasPrefix(requestPath, "/request") ||
				strings.HasPrefix(requestPath, "/code-server/request") {
				return next(c)
			}
			username, sessionID, groups, ok, err := authService.IsLoggedIn(c)
			//log.Info().Msgf("%s - %s -%v - %v, %v", username, sessionID, groups, ok, err)
			if ok {
				if len(requireGroups) > 0 && !authService.HasAnyRequiredGroup(groups, requireGroups) {
					msg := "forbidden!"
					if errorHeader != "" {
						c.Request().Header.Set(errorHeader, msg)
					}
					log.Warn().
						Str("username", username).
						Strs("groups", groups).
						Strs("required", requireGroups).
						Str("path", requestPath).
						Msg("group requirement failed")

					return c.NoContent(http.StatusForbidden)
				}
				c.Set("username", username)
				c.Set("groups", groups)
				c.Set("sessionID", sessionID)
				return next(c)
			} else if errorHeader != "" {
				if err != nil {
					c.Request().Header.Set(errorHeader, err.Error())
				}
			}

			//log.Info().Msgf("%v")
			return c.String(http.StatusForbidden, "access denied")
		}
	}
}

func agentKeyMiddleware(expectedKey string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			agentKey := c.Request().Header.Get("X-Agent-Key")

			if strings.HasPrefix(c.Request().URL.Path, "/code-server/request") ||
				strings.HasPrefix(c.Request().URL.Path, "/request") {
				return next(c)
			}

			if agentKey == "" {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "missing X-Agent-Key header",
				})
			}
			if agentKey != expectedKey {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "invalid X-Agent-Key",
				})
			}
			return next(c)
		}
	}
}

func RegisterRoutes(log zerolog.Logger, config *config.Config) (*echo.Echo, *xdiscovery.Agent) {

	e := echo.New()

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			"*",
		},
		AllowHeaders: []string{
			"*",
		},
	}))
	e.Use(middleware.Logger())

	agentKeyMiddlewareForAPI := agentKeyMiddleware(config.AgentMetadata.AgentKey)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	// Agent
	containerService := service.NewContainerService(cli, config, log)
	containerHandler := handlers.NewContainerHandler(containerService)
	metricsService := service.NewMetricsService(log, config)
	metricsHandler := handlers.NewMetricsHandler(metricsService)
	restClient := adapters.NewRestyClientAdapter()
	discoveryService := service.NewDiscoveryService(restClient, config, log)
	agent := xdiscovery.NewAgent(config, discoveryService, time.Second*10, log)
	agentHandler := handlers.NewAgentHandler(config)

	// Proxy Config

	// Redis Client
	rdbOt := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port),
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
		PoolSize: 1000,
	}
	redisClient, err := xsession.NewRedisClient(rdbOt)
	if err != nil {
		panic(err)
	}

	// Session Driver w/redis
	sessionSecret := utils.NewEncKey32FromSecret(config.Secrets.SessionSecret)
	sessionDriver, err := xsession.NewRedisSessionManager(redisClient, "session", sessionSecret, log)
	if err != nil {
		panic(err)
	}

	// Proxy Common
	inMemCache := inmemory.NewInMemoryCache()
	dummyProxy := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: "localhost"})
	proxyService := service.NewProxyService(log)
	jwtService := security.NewJWTService(
		config.Secrets.JWTAccessKey,
		config.Secrets.JWTRefreshKey,
		config.Secrets.JWTIssuer,
		config.Secrets.JWTAudience,
		log)
	authService := service.NewAuthService(jwtService, false, log, sessionDriver)
	proxyHandler := handlers.NewProxyHandler(
		dummyProxy,
		proxyService,
		inMemCache,
		log,
		"code-server",
		8443,
		true,
		false,
	)

	// /api/v1
	apiGroup := e.Group("/api/v1", agentKeyMiddlewareForAPI)
	apiGroup.POST("/containers", containerHandler.CreateContainer)
	apiGroup.POST("/containers/:id/start", containerHandler.StartContainer)
	apiGroup.POST("/containers/:id/stop", containerHandler.StopContainer)
	apiGroup.POST("/containers/:id/restart", containerHandler.RestartContainer)
	apiGroup.DELETE("/containers/:id", containerHandler.RemoveContainer)
	apiGroup.POST("/containers/:id", containerHandler.RemoveContainer)
	apiGroup.GET("/containers", containerHandler.ListContainers)
	apiGroup.GET("/containers/code-server", containerHandler.ListCodeServerContainers)
	apiGroup.GET("/containers/:id/logs", containerHandler.LogsContainer)
	apiGroup.GET("/containers/:name/id", containerHandler.GetContainerIDByName)
	apiGroup.GET("/containers/defaults", containerHandler.GetConfigDefaultsHandler)
	apiGroup.GET("/containers/:name/exist", containerHandler.IsContainerExistHandler)
	apiGroup.GET("/containers/:name/running", containerHandler.IsContainerRunningHandler)
	apiGroup.GET("/containers/:name/stats", containerHandler.GetContainerStats)
	apiGroup.GET("/metrics", metricsHandler.Fetch)
	apiGroup.GET("/tags", agentHandler.GetTags)

	// /code-server
	e.Any("/code-server/*",
		proxyHandler.EchoHandler(),
		JWTAuthMiddleware(authService, "X-Proxy-Error", false, log, []string{}),
		agentKeyMiddlewareForAPI,
	)

	return e, agent
}
