package routes

import (
	"context"
	"fmt"
	"html/template"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"v0/internal/app/adapters"
	"v0/internal/app/api/handlers"
	"v0/internal/app/api/middleware"
	"v0/internal/app/security"
	"v0/internal/app/service"
	"v0/internal/app/xdiscovery"
	"v0/internal/app/xsession"
	"v0/internal/config"
	"v0/internal/utils"
)

func RegisterRoutes(log zerolog.Logger, config *config.AppConfig, tmpl *template.Template) *echo.Echo {

	e := echo.New()

	middleware.SetPreMiddlewares(e, log, config)

	// Auth Strategy: LDAP
	if strings.ToLower(config.AuthBackend) == "ldap" {
		log.Info().Msg("Configured auth strategy: LDAP")

		security.InitLDAP(config, log)

		var ldapScheme string
		switch config.AuthLdapPort {
		case "389":
			ldapScheme = "ldap"
		case "636":
			ldapScheme = "ldaps"
		default:
			ldapScheme = "ldaps"
		}

		ldapUrl := fmt.Sprintf("%s://%s:%s", ldapScheme, config.AuthLdapServer, config.AuthLdapPort)
		l, err := ldap.DialURL(ldapUrl)
		if err != nil {
			panic(err)
		}
		defer l.Close()

		err = l.Bind(config.AuthLdapBindUser, config.AuthLdapBindPassword)
		if err != nil {
			panic(err)
		}

		log.Info().Msg("LDAP Connection OK")
	} else {
		panic("Only LDAP Auth Supported.")
	}

	regularRoles := utils.ParseToList(config.AuthRegularRoles)
	adminRoles := utils.ParseToList(config.AuthAdminRoles)

	rdbOt := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.RedisHost, config.RedisPort),
		Password: config.RedisPassword,
		DB:       config.RedisDB,
		PoolSize: 1000,
	}
	redisClient, err := xsession.NewRedisClient(rdbOt)
	if err != nil {
		panic(err)
	}
	sessionSecret := utils.NewEncKey32FromSecret(config.AppSessionSecret)
	sessionDriver, err := xsession.NewRedisSessionManager(redisClient, "session", sessionSecret, log)
	if err != nil {
		panic(err)
	}

	jwtService := security.NewJWTService(config.JWTAccessSecret, config.JWTRefreshSecret, config.JWTIssuer, config.JWTAudience, log)
	authService := service.NewAuthService(jwtService, tmpl, config.AppWithTLS, log, sessionDriver, config.AppSessionCookie)
	notFoundPageService := service.NewNotFoundPageService(tmpl)

	revoker := xsession.NewRevoker(sessionDriver, log, config.AppSessionCookie)
	codeServerSessions := xsession.NewSessionRegistry(config.AppWithTLS, revoker, log)
	restyAdapter := adapters.NewRestyClientAdapter()
	agentService := service.NewAgentService(restyAdapter, log, config.AppAgentKey, redisClient)
	containerRegService := service.NewContainerRegistryService(redisClient, log)

	agentKeyMiddleware := middleware.AgentKeyMiddleware(config.AppAgentKey, log)
	csrfMiddleware := middleware.CustomCSRFMiddleware(config.AppWithTLS, "form:_csrf")
	jwtMiddlewareForUsers := middleware.JWTAuthMiddleware(authService, "", revoker, config.AppWithTLS, log, regularRoles)
	jwtMiddlewareForAdmins := middleware.JWTAuthMiddleware(authService, "", revoker, config.AppWithTLS, log, adminRoles)
	jwtMiddlewareForProxy := middleware.JWTAuthMiddleware(authService, "X-Proxy-Error", revoker, config.AppWithTLS, log, regularRoles)
	jwtMiddlewareForProxyAdmins := middleware.JWTAuthMiddleware(authService, "X-Proxy-Error", revoker, config.AppWithTLS, log, adminRoles)

	standardCORSMiddleware := middleware.StandardCORSMiddleware(config, []string{})

	errorHandlerSvc := service.NewErrorHandlerService("/csplatform/404")
	e.HTTPErrorHandler = errorHandlerSvc.GlobalHTTPErrorHandler()

	containerHandler := handlers.NewContainerHandler(tmpl, agentService, log, containerRegService, config)

	// /api/v1
	codeServerSessionHandler := handlers.NewCodeServerSessionHandler(codeServerSessions, tmpl)
	apiGroup := e.Group("/api/v1", jwtMiddlewareForAdmins, standardCORSMiddleware)
	apiGroup.GET("/sessions", codeServerSessionHandler.ListSessions)
	apiGroup.POST("/sessions/conns", codeServerSessionHandler.ListConnectionsPost)
	apiGroup.DELETE("/sessions/conns", codeServerSessionHandler.CancelConnPost)
	apiGroup.DELETE("/sessions", codeServerSessionHandler.CancelAllPost)
	apiGroup.DELETE("/sessions/idle", codeServerSessionHandler.CloseIdlePost)

	//
	apiGroup.POST("/containers/create", containerHandler.CreateContainerRequest)

	// /admin
	adminGroup := e.Group("/admin", csrfMiddleware, jwtMiddlewareForAdmins, standardCORSMiddleware)
	adminGroup.GET("/code-server-sessions", codeServerSessionHandler.RenderPage)

	// /csplatform
	homePageHandler := handlers.NewHomePageHandler(jwtService, tmpl, config, log, agentService, containerRegService)
	notFoundHandler := handlers.NewNotFoundPageHandler(tmpl)
	csplatformGroup := e.Group("/csplatform", csrfMiddleware, jwtMiddlewareForUsers, standardCORSMiddleware)
	csplatformGroup.GET("/home", homePageHandler.RenderHomePage)
	csplatformGroup.GET("/404", notFoundHandler.Render404)
	csplatformGroup.GET("/containers/create", containerHandler.ShowFormCreate)
	csplatformGroup.POST("/containers/stop", containerHandler.StopContainer)
	csplatformGroup.POST("/containers/restart", containerHandler.RestartContainer)
	csplatformGroup.POST("/containers/start", containerHandler.StartContainer)
	csplatformGroup.POST("/containers/delete", containerHandler.RemoveContainer)

	// /discovery
	discoveryRegistry := xdiscovery.NewRegistry(redisClient, time.Second*30, log)
	discoveryHandler := handlers.NewDiscoveryHandler(discoveryRegistry, log)
	discoveryGroup := e.Group("/discovery", agentKeyMiddleware)
	discoveryGroup.POST("/register", discoveryHandler.Register)
	discoveryGroup.POST("/deregister", discoveryHandler.Discover)
	discoveryGroup.POST("/healthcheck", discoveryHandler.HealthCheck)
	discoveryGroup.GET("/discover/:serviceName", discoveryHandler.Discover)

	// /auth
	authHandler := handlers.NewAuthHandler(authService, codeServerSessions, tmpl, config.AppWithTLS, log)
	authGroup := e.Group("/auth", csrfMiddleware, standardCORSMiddleware)
	authGroup.GET("/login", authHandler.GetLogin)
	authGroup.POST("/login", authHandler.PostLogin)
	authGroup.GET("/logout", authHandler.PostLogout, jwtMiddlewareForUsers)
	authGroup.POST("/logout", authHandler.PostLogout, jwtMiddlewareForUsers)

	proxyService := service.NewProxyService(notFoundPageService, log)

	// /redisinsight
	if config.RedisInsightEnabled {
		redisInsightProxy := httputil.NewSingleHostReverseProxy(&url.URL{
			Scheme: config.RedisInsightProto,
			Host:   config.RedisInsightURL,
		})
		rih := handlers.NewRedisInsightProxyHandler(
			redisInsightProxy,
			proxyService,
			jwtService,
			log,
			config,
		)
		riGroup := e.Group("/redisinsight")
		riGroup.Any("/ui/*", rih.EchoHandler(), jwtMiddlewareForProxyAdmins, standardCORSMiddleware)
	}

	// /
	dummyProxy := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: "localhost"})
	ph := handlers.NewProxyHandler(
		config.AppAgentKey,
		dummyProxy,
		proxyService,
		containerRegService,
		jwtService,
		log,
		config.AppWithTLS,
	)
	e.Any("/code-server/*", ph.EchoHandler(codeServerSessions), jwtMiddlewareForProxy)

	janitorCtx, _ := context.WithCancel(context.Background())
	codeServerSessions.StartJanitor(janitorCtx, 10*time.Second)

	return e
}
