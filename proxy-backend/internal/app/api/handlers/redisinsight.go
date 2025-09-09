package handlers

import (
	"net/http"
	"net/http/httputil"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"v0/internal/app/security"
	"v0/internal/app/service"
	"v0/internal/config"
	"v0/internal/utils"
)

type RedisInsightProxyHandler struct {
	config       *config.AppConfig
	proxy        *httputil.ReverseProxy
	proxyService *service.ProxyService
	jwtService   *security.JWTService
	log          zerolog.Logger
	withTLS      bool
}

func NewRedisInsightProxyHandler(
	proxy *httputil.ReverseProxy,
	proxyService *service.ProxyService,
	jwtService *security.JWTService,
	log zerolog.Logger,
	config *config.AppConfig,
) *RedisInsightProxyHandler {
	return &RedisInsightProxyHandler{config, proxy, proxyService, jwtService, log, false}
}

func (h *RedisInsightProxyHandler) EchoHandler() echo.HandlerFunc {
	return func(c echo.Context) error {

		baseTransport := h.proxyService.BaseTransportInit(false)

		rp := *h.proxy
		rp.Transport = baseTransport
		globalReq := c.Request()
		h.proxyService.SetProxyErrorHandler(&rp, c)

		rp.ModifyResponse = func(resp *http.Response) error {
			path := resp.Request.URL.Path

			if mimeType := utils.GetMimeTypeFromUrlSuffix(path); mimeType != "" {
				resp.Header.Set("Content-Type", mimeType)
			}

			return nil
		}

		rp.Director = func(req *http.Request) {
			req.URL.Scheme = h.config.RedisInsightProto
			req.URL.Host = h.config.RedisInsightURL
			req.Header.Set("X-Real-IP", c.RealIP())
			req.Header.Set("X-Forwarded-For", req.RemoteAddr)
		}
		rp.ServeHTTP(c.Response().Writer, globalReq)
		return nil
	}
}
