package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"v0/internal/app/security"
	"v0/internal/app/service"
	"v0/internal/app/xsession"
	"v0/internal/utils"
)

func getSessionID(c echo.Context) string {
	if v := c.Get("username"); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return "u:" + s
		}
	}
	return ""
}

type ProxyHandler struct {
	agentKey     string
	proxy        *httputil.ReverseProxy
	proxyService *service.ProxyService
	reg          *service.ContainerRegistryService
	jwtService   *security.JWTService
	log          zerolog.Logger
	withTLS      bool
}

func NewProxyHandler(
	agentKey string,
	proxy *httputil.ReverseProxy,
	proxyService *service.ProxyService,
	reg *service.ContainerRegistryService,
	jwtService *security.JWTService,
	log zerolog.Logger,
	withTLS bool) *ProxyHandler {
	return &ProxyHandler{agentKey, proxy, proxyService, reg, jwtService, log, withTLS}
}

func (h *ProxyHandler) EchoHandler(CodeServerSessionRegistry *xsession.CodeServerSessionRegistry) echo.HandlerFunc {
	return func(c echo.Context) error {

		baseTransport := h.proxyService.BaseTransportInit(true)

		rp := *h.proxy
		rp.Transport = baseTransport
		globalReq := c.Request()

		var redisSessionID string
		redisSessionIDCtx, ok := c.Get("sessionID").(string)
		if ok {
			redisSessionID = redisSessionIDCtx
		}

		if strings.Contains(c.QueryString(), "reconnectionToken") && strings.Contains(c.QueryString(), "skipWebSocketFrames") {

			sessionID := getSessionID(c)

			ctx, cancel := context.WithCancel(c.Request().Context())
			globalReq = c.Request().Clone(ctx)

			connID := fmt.Sprintf("%s | %s", redisSessionID, c.QueryString())
			CodeServerSessionRegistry.AddConn(sessionID, connID, ctx, cancel, baseTransport)
		}

		h.proxyService.SetProxyErrorHandler(&rp, c)

		rp.ModifyResponse = func(resp *http.Response) error {
			path := resp.Request.URL.Path
			status := resp.StatusCode

			if status == http.StatusNotFound &&
				!strings.HasSuffix(resp.Request.URL.Path, "vsda.js") &&
				!strings.HasSuffix(resp.Request.URL.Path, "vsda_bg.wasm") {
				//resp.StatusCode = http.StatusFound
				//resp.Header.Set("Location", "/csplatform/404")
				//resp.Body = io.NopCloser(strings.NewReader(""))
				//resp.Header.Del("Content-Length")
				return nil
			}

			if mimeType := utils.GetMimeTypeFromUrlSuffix(path); mimeType != "" {
				resp.Header.Set("Content-Type", mimeType)
			}

			parts := strings.Split(strings.Trim(path, "/"), "/")
			if len(parts) >= 4 && parts[0] == "request" {
				protocol := strings.ToLower(parts[2])
				if protocol == "sse" || protocol == "sse-https" {
					resp.Header.Set("Content-Type", "text/event-stream")
					resp.Header.Set("Cache-Control", "no-cache")
					resp.Header.Set("Connection", "keep-alive")
				}
			}
			return nil
		}

		rp.Director = func(req *http.Request) {

			ctx := context.Background()
			info, err := h.reg.Get(ctx, c.Get("username").(string))
			if err != nil {
				return
			}
			targetHost := info.AgentHost
			targetHost = strings.TrimPrefix(targetHost, "http://")
			targetHost = strings.TrimPrefix(targetHost, "https://")

			targetURL := &url.URL{
				Scheme: "http",
				Host:   targetHost,
			}
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.Header.Set("Host", targetHost)
			req.Header.Set("X-Real-IP", req.RemoteAddr)
			req.Header.Set("X-Forwarded-For", req.RemoteAddr)
			req.Header.Set("X-Forwarded-Proto", "http")
			req.Header.Set("User-Agent", c.Request().Header.Get("User-Agent"))
			req.Header.Set("X-Agent-Key", h.agentKey)
			if redisSessionID != "" {
				req.Header.Set("X-Session-ID", redisSessionID)
			}

		} //end func
		rp.ServeHTTP(c.Response().Writer, globalReq)
		return nil
	}
}

func (h *ProxyHandler) setCodeServerProxyHeaders(req *http.Request, targetURL *url.URL, reqPath, originalPath string) {
	req.URL.Scheme = targetURL.Scheme
	req.URL.Host = targetURL.Host
	req.URL.Path = reqPath
	req.Header.Set("Host", targetURL.Host)
	req.Header.Set("X-Real-IP", req.RemoteAddr)
	req.Header.Set("X-Forwarded-For", req.RemoteAddr)
	req.Header.Set("X-Original-Path", originalPath)
	if h.withTLS {
		req.Header.Set("X-Forwarded-Proto", "https")
	} else {
		req.Header.Set("X-Forwarded-Proto", "http")
	}
	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", req.Header.Get("Upgrade"))
	req.Header.Set("Accept-Encoding", "gzip")

}
