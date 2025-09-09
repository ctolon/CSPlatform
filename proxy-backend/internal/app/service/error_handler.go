package service

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

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
		"favicon.ico",
	}
)

type ErrorHandlerService struct {
	NotFoundURL string
}

func NewErrorHandlerService(notFoundURL string) *ErrorHandlerService {
	return &ErrorHandlerService{notFoundURL}
}

func (s *ErrorHandlerService) isJSONRequest(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(strings.ToLower(accept), "application/json")
}

func (s *ErrorHandlerService) GlobalHTTPErrorHandler() echo.HTTPErrorHandler {

	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		path := c.Request().URL.Path

		for _, p := range allowPrefixes {
			if strings.HasPrefix(path, p) {
				return
			}
		}
		for _, sfx := range allowSuffixes {
			if strings.HasSuffix(path, sfx) {
				return
			}
		}

		req := c.Request()
		url := req.URL.String()
		method := req.Method
		remoteIP := c.RealIP()
		c.Logger().Error("Error! Method: ", method, " ", "URL: ", url, " ", "RemoteIP: ", remoteIP, " ", "Error: ", err)

		code := http.StatusInternalServerError
		msg := "Internal Server Error"
		var he *echo.HTTPError
		if errors.As(err, &he) {
			code = he.Code
			if he.Message != nil {
				msg = fmt.Sprintf("%v", he.Message)
			} else {
				msg = http.StatusText(code)
			}
		}

		if code == http.StatusNotFound {
			c.Logger().Warnf("404 Not Found :%s", c.Request().RequestURI)

			if s.isJSONRequest(c.Request()) || c.Request().Method == http.MethodPost || c.Request().Method == http.MethodPut {
				_ = c.JSON(http.StatusNotFound, map[string]any{
					"error":   "not found",
					"message": fmt.Sprintf("The resources '%s' was not found", c.Request().RequestURI),
				})
				return
			}

			if c.Request().Method == http.MethodGet || c.Request().Method == http.MethodHead {
				c.Redirect(http.StatusFound, s.NotFoundURL)
			}

			_ = c.String(http.StatusNotFound, "404 Not Found")
			return
		}

		_ = c.JSON(code, map[string]any{
			"error":   code,
			"message": msg,
		})
	}
}
