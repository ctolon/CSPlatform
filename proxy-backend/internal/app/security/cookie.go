package security

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

func ExpireSessionCookie(c echo.Context, withTLS bool, sessionCookie string) {
	del := &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   withTLS,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
	c.SetCookie(del)

	del2 := *del

	del2.Path = c.Request().URL.Path
	c.SetCookie(&del2)
}
