package service

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"a0/internal/app/constants"
)

type ProxyService struct {
	//notFoundService *NotFoundPageService
	log zerolog.Logger
}

func NewProxyService(log zerolog.Logger) *ProxyService {
	return &ProxyService{log}
}

func (s *ProxyService) SetProxyErrorHandler(rp *httputil.ReverseProxy, c echo.Context) {
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {

		/*
			username := ""
			val := c.Get("username")
			uname, ok := val.(string)
			if ok && uname != "" {
				username = uname
			}

		*/
		if proxyErr := r.Header.Get("X-Proxy-Error"); proxyErr != "" {

			if strings.HasSuffix(err.Error(), "no such host") {
				c.JSON(http.StatusBadGateway, map[string]interface{}{
					"error": "no such host",
					"path":  r.URL.Path,
					"host":  r.URL.Host,
					//"username":  username,
				})
				return
			}

			if strings.HasSuffix(err.Error(), "connection refused") {
				c.JSON(http.StatusBadGateway, map[string]interface{}{
					"error": "connection refused",
					"path":  r.URL.Path,
					"host":  r.URL.Host,
					//"username":  username,
				})
				return
			}

			s.log.Err(err).Msg("proxy error - 001")
			status := http.StatusBadRequest
			w.WriteHeader(status)
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]string{
				"error":  "proxy backend request validation failed",
				"detail": proxyErr,
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.HasSuffix(err.Error(), "no such host") {
			c.JSON(http.StatusBadGateway, map[string]interface{}{
				"error": "no such host",
				"path":  r.URL.Path,
				"host":  r.URL.Host,
				//"username":  username,
			})
			return
		}

		if strings.HasSuffix(err.Error(), "connection refused") {
			c.JSON(http.StatusBadGateway, map[string]interface{}{
				"error": "connection refused",
				"path":  r.URL.Path,
				"host":  r.URL.Host,
				//"username":  username,
			})
			return
		}

		s.log.Error().Err(err).Msg("proxy error - 002")
		status := http.StatusBadGateway
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		resp := map[string]string{
			"error":  "proxy backend connection failed",
			"detail": err.Error(),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (h *ProxyService) RequestToFmt(scheme string, host string, path string) string {
	return fmt.Sprintf("%s://%s%s", scheme, host, path)
}

func (h *ProxyService) BaseTransportInit(base bool) *http.Transport {

	if base {
		return &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS12,
				MaxVersion:         tls.VersionTLS12,
				CipherSuites:       constants.TLSCiphers,
			},
			DialContext: (&net.Dialer{
				Timeout:   120 * time.Second,
				KeepAlive: 120 * time.Second,
			}).DialContext,
			TLSNextProto:        make(map[string]func(string, *tls.Conn) http.RoundTripper),
			MaxIdleConns:        100,
			IdleConnTimeout:     600 * time.Second,
			TLSHandshakeTimeout: 20 * time.Second,
		}
	}
	return &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
			MaxVersion:         tls.VersionTLS12,
			CipherSuites:       constants.TLSCiphers,
		},
		DialContext: (&net.Dialer{
			Timeout:   120 * time.Second,
			KeepAlive: 120 * time.Second,
		}).DialContext,
		TLSNextProto:          make(map[string]func(string, *tls.Conn) http.RoundTripper),
		MaxIdleConns:          100,
		IdleConnTimeout:       600 * time.Second,
		TLSHandshakeTimeout:   20 * time.Second,
		ResponseHeaderTimeout: 900 * time.Second,
	}
}
