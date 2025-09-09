package handlers

import (
	//"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"a0/internal/app/inmemory"
	"a0/internal/app/service"
	"a0/internal/app/utils"
	"a0/internal/app/xerror"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

type ProxyHandler struct {
	proxy         *httputil.ReverseProxy
	proxyService  *service.ProxyService
	inMemoryCache *inmemory.InMemoryCache
	log           zerolog.Logger
	Host          string
	Port          int
	WithUsername  bool
	withTLS       bool
}

func NewProxyHandler(
	proxy *httputil.ReverseProxy,
	proxyService *service.ProxyService,
	inmemoryCache *inmemory.InMemoryCache,
	log zerolog.Logger,
	host string,
	port int,
	withUsername bool,
	withTLS bool) *ProxyHandler {
	return &ProxyHandler{proxy, proxyService, inmemoryCache, log, host, port, withUsername, withTLS}
}

func (h *ProxyHandler) EchoHandler() echo.HandlerFunc {
	return func(c echo.Context) error {

		baseTransport := h.proxyService.BaseTransportInit(true)

		rp := *h.proxy
		rp.Transport = baseTransport
		globalReq := c.Request()

		h.proxyService.SetProxyErrorHandler(&rp, c)

		rp.ModifyResponse = func(resp *http.Response) error {
			path := resp.Request.URL.Path
			status := resp.StatusCode

			if status == http.StatusNotFound &&
				!strings.HasSuffix(resp.Request.URL.Path, "vsda.js") &&
				!strings.HasSuffix(resp.Request.URL.Path, "vsda_bg.wasm") {
				resp.Body = io.NopCloser(strings.NewReader(""))
				resp.Header.Del("Content-Length")
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

			// remove /code-server prefix
			if strings.HasPrefix(req.URL.Path, "/code-server/") {
				req.URL.Path = strings.TrimPrefix(req.URL.Path, "/code-server")
			}

			parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
			// h.log.Debug().Msgf("parts: %v", parts)
			_scheme := "http"

			if len(parts) == 2 && parts[0] == "update" && parts[1] == "check" {
				username := c.Get("username").(string)
				req.URL.Scheme = _scheme
				req.URL.Host = h.setHost(username)
				req.URL.Path = "/update/check"
				return
			}
			if len(parts) == 1 && parts[0] == "mint-key" {
				username := c.Get("username").(string)
				req.URL.Scheme = _scheme
				req.URL.Host = h.setHost(username)
				req.URL.Path = "/mint-key"
				return
			}

			if len(parts) >= 4 && parts[0] == "request" {
				username := strings.ToLower(parts[1])
				protocol := strings.ToLower(parts[2])
				portStr := parts[3]

				switch protocol {
				case "http", "https", "sse", "sse-https":
				default:
					err := &xerror.ErrInvalidProtocol{}
					req.URL.Path = ""
					req.Body = io.NopCloser(strings.NewReader(err.Error()))
					req.Header.Set("X-Proxy-Error", err.Error())
					return
				}
				portInt, err := strconv.Atoi(portStr)
				if err != nil {
					err := &xerror.ErrInvalidPortNumberCode1{}
					req.URL.Path = ""
					req.Body = io.NopCloser(strings.NewReader(err.Error()))
					req.Header.Set("X-Proxy-Error", err.Error())
					return
				}
				if !h.inMemoryCache.IsValidPort(portInt) {
					err := &xerror.ErrInvalidPortNumberCode2{}
					req.URL.Path = ""
					req.Body = io.NopCloser(strings.NewReader(err.Error()))
					req.Header.Set("X-Proxy-Error", err.Error())
					return
				}

				extraPath := ""
				if len(parts) > 4 {
					extraPath = "/" + strings.Join(parts[4:], "/")
				}

				isSSE := false
				if protocol == "sse" {
					protocol = "http"
					isSSE = true
				}
				if protocol == "sse-https" {
					protocol = "https"
					isSSE = true
				}

				req.URL.Scheme = protocol
				req.URL.Host = h.setHostForProxy(username, portStr)
				req.URL.Path = extraPath
				req.Header.Set("X-Real-IP", c.RealIP())
				req.Header.Set("X-Forwarded-For", req.RemoteAddr)
				req.Header.Set("X-Forwareded-Proto", protocol)
				req.Header.Set("Accept", "application/json")
				req.Header.Add("Content-TYpe", "application/json")
				if isSSE {
					req.Header.Set("Connection", "keep-alive")
					req.Header.Set("Accept", "application/json, text/event-stream")
				}
				return
			}

			if len(parts) == 1 && !strings.HasPrefix(req.URL.Path, "/manifest.json") && !strings.HasPrefix(req.URL.Path, "/request") {
				username := c.Get("username").(string)
				targetURL := &url.URL{
					Scheme: _scheme,
					Host:   h.setHost(username),
				}
				// h.log.Debug().Msgf("%s", targetURL.Host)
				reqPath := "/" + strings.Join(parts[1:], "/")
				// msg := h.requestToFmt(targetURL.Scheme, targetURL.Host, reqPath)
				// h.log.Debug().Msgf("Request To: %s", msg)
				h.setCodeServerProxyHeaders(req, targetURL, reqPath, req.URL.Path)
				return
			} else {
				uname := c.Get("username")
				username, _ := uname.(string)
				targetURL := &url.URL{}
				if h.WithUsername {
					if strings.HasPrefix(req.URL.Path, "/_static") || strings.HasPrefix(req.URL.Path, "/manifest.json") {
						req.URL.Scheme = _scheme
						req.URL.Host = h.setHost(username)
						return
					}
					targetURL.Scheme = _scheme
					targetURL.Host = h.setHost(username)
					reqPath := "/" + strings.Join(parts[1:], "/")
					//msg := h.requestToFmt(targetURL.Scheme, targetURL.Host, reqPath)
					//h.log.Debug().Msgf("Request to: %s", msg)
					h.setCodeServerProxyHeaders(req, targetURL, reqPath, req.URL.Path)
					return
					// TODO remove else cond
				} else {
					req.URL.Path = ""
					req.Body = io.NopCloser(strings.NewReader("invalid creds"))
					req.Header.Set("X-Proxy-Error", "invalid creds")
					return
				} // endif
			} // endif
		} //end func
		rp.ServeHTTP(c.Response().Writer, globalReq)
		return nil
	}
}

func (h *ProxyHandler) setHost(username string) string {
	if h.WithUsername {
		return fmt.Sprintf("%s-%s:%d", h.Host, username, h.Port)
	}
	return fmt.Sprintf("%s:%d", h.Host, h.Port)
}

func (h *ProxyHandler) setHostForProxy(username, port string) string {
	if h.WithUsername {
		return fmt.Sprintf("%s-%s:%s", h.Host, username, port)
	}
	return fmt.Sprintf("%s:%s", h.Host, port)
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
