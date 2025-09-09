package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	"v0/internal/app/service"
	"v0/internal/app/xsession"
	"v0/internal/config"
)

func SetPreMiddlewares(e *echo.Echo, log zerolog.Logger, cfg *config.AppConfig) {
	e.IPExtractor = echo.ExtractIPDirect()
	e.Use(middleware.RequestID())
	e.Use(ZerologMiddleware(log, cfg.AppSessionCookie))
	e.Use(middleware.Recover())
	if cfg.AppWithTLS {
		e.Use(middleware.Secure())
		e.Pre(middleware.HTTPSRedirect())
	}
}

func CustomCSRFMiddleware(withTLS bool, tokenlookup string) echo.MiddlewareFunc {
	return middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup:  tokenlookup,
		CookieSecure: withTLS,
		CookiePath:   "/",
	})
}

var (
	allowPrefixes = []string{
		"/_static",
		"/stable-",
		"/manifest.json",
		"/update/check",
		"/redisinsight/ui",
		"/mint",
	}
	allowSuffixes = []string{
		"vsda.js",
		"vsda_bg.wasm",
	}
)

// ZerologMiddleware Custom Logger Middleware which uses zerolog
func ZerologMiddleware(logger zerolog.Logger, sessionCookie string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			if hasAnyPrefix(c.Request().URL.Path, allowPrefixes) ||
				hasAnySuffix(c.Request().URL.Path, allowSuffixes) {
				return next(c)
			}
			uname := "anonym"
			sess := "not exist"
			username, ok := c.Get("username").(string)
			if ok && username != "" {
				uname = username
			}
			sessionID, cerr := c.Cookie(sessionCookie)
			if cerr == nil && sessionID != nil && sessionID.Value != "" {
				sess = sessionID.Value
			}

			start := time.Now()
			err := next(c)
			stop := time.Now()
			req := c.Request()
			res := c.Response()
			latency := stop.Sub(start)
			logger.Info().
				Str("method", req.Method).
				Str("uri", req.RequestURI).
				Str("host", req.Host).
				Str("user_id", uname).
				Str("session_id", sess).
				Int("status", res.Status).
				Str("remote_ip", c.RealIP()).
				Dur("latency", latency).
				Str("latency_human", latency.String()).
				Str("user_agent", req.UserAgent()).
				Msg("log on request")
			return err
		}
	}
}

// Middleware for JWT Authentication for login
func JWTAuthMiddleware(
	authService *service.AuthService,
	errorHeader string,
	revoker *xsession.Revoker,
	withTLS bool,
	log zerolog.Logger,
	requireGroups []string,
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			requestPath := c.Request().URL.Path
			// log.Info().Msgf("%s", requestPath)
			username, sessionID, groups, ok, err := isLoggedInWebSocketGuard(c, revoker, withTLS, authService, log)
			// log.Info().Msgf("%s - %s -%v - %v, %v", username, sessionID, groups, ok, err)
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

			if strings.HasPrefix(requestPath, "/request") {
				return next(c)
			}

			if requestPath == "/csplatform/404" {
				return next(c)
			}
			//log.Info().Msgf("%v")
			return c.Redirect(http.StatusFound, "/auth/login")
		}
	}
}

func isLoggedInWebSocketGuard(
	c echo.Context,
	revoker *xsession.Revoker,
	withTLS bool,
	authService *service.AuthService,
	log zerolog.Logger,
) (string, string, []string, bool, error) {
	if strings.Contains(c.QueryString(), "reconnectionToken") && strings.Contains(c.QueryString(), "skipWebSocketFrames") {
		// log.Debug().Msgf("Websocket request")
		return revoker.ConsumeIfRevokedWithLoggedIn(c, withTLS, authService.IsLoggedIn)
	} else {
		return authService.IsLoggedIn(c)
	}
}

func StandardCORSMiddleware(cfg *config.AppConfig, allowMethods []string) echo.MiddlewareFunc {

	allowOrigins := strings.Split(cfg.AppCORS, ",")
	if len(allowMethods) == 0 {
		allowMethods = []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
		}
	}

	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: allowOrigins,
		AllowMethods: allowMethods,
	})
}

func CustomCORSMiddleware(allowOrigins []string, allowMethods []string) echo.MiddlewareFunc {

	if len(allowOrigins) == 0 {
		allowOrigins = []string{"*"}
	}

	if len(allowMethods) == 0 {
		allowMethods = []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
		}
	}

	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: allowOrigins,
		AllowMethods: allowMethods,
	})

}

func AgentKeyMiddleware(expectedKey string, log zerolog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			agentKey := c.Request().Header.Get("X-Agent-Key")
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

func hasAnyPrefix(s string, prefs []string) bool {
	for _, p := range prefs {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func hasAnySuffix(s string, sufs []string) bool {
	for _, x := range sufs {
		if strings.HasSuffix(s, x) {
			return true
		}
	}
	return false
}
