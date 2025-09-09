package handlers

import (
	"context"
	"html/template"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/shaj13/go-guardian/v2/auth"

	"v0/internal/app/security"
	"v0/internal/app/service"
	"v0/internal/app/xsession"
)

type AuthHandler struct {
	authService               *service.AuthService
	codeServerSessionRegistry *xsession.CodeServerSessionRegistry
	tmpl                      *template.Template
	withTLs                   bool
	log                       zerolog.Logger
}

func NewAuthHandler(
	authService *service.AuthService,
	codeServerSessionRegistry *xsession.CodeServerSessionRegistry,
	tmpl *template.Template,
	withTLs bool,
	log zerolog.Logger,
) *AuthHandler {
	return &AuthHandler{authService, codeServerSessionRegistry, tmpl, withTLs, log}
}

func (h *AuthHandler) GetLogin(c echo.Context) error {

	_, _, _, isLoggedIn, err := h.authService.IsLoggedIn(c)
	if isLoggedIn {
		return c.Redirect(http.StatusFound, "/csplatform/home")
	}
	if err.Error() == "sid not found" {
		security.ExpireSessionCookie(c, h.withTLs, h.authService.SessionCookie)
		return c.Redirect(http.StatusTemporaryRedirect, "/auth/login")
	}
	data := make(map[string]any)
	data["CSRFToken"] = c.Get("csrf").(string)
	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.tmpl.ExecuteTemplate(c.Response(), "login.go.tmpl", data)
}

func (h *AuthHandler) PostLogin(c echo.Context) error {

	ip := c.RealIP()
	ua := c.Request().Header.Get("User-Agent")
	requireGroups := []string{"bdadmins", "bddataengineers"}

	_, _, _, isLoggedIn, err := h.authService.IsLoggedIn(c)
	if isLoggedIn {
		return c.Redirect(http.StatusFound, "/csplatform/home")
	}
	if err.Error() == "sid not found" {
		security.ExpireSessionCookie(c, h.withTLs, h.authService.SessionCookie)
		return c.Redirect(http.StatusTemporaryRedirect, "/auth/login")
	}
	username := c.FormValue("username")
	password := c.FormValue("password")
	req := c.Request()
	req.SetBasicAuth(username, password)
	user, err := security.AuthStrategy.Authenticate(c.Request().Context(), &http.Request{
		Header: c.Request().Header,
	})
	if err != nil || user == nil || user.GetUserName() != username {
		c.Logger().Error(err.Error())
		data := map[string]any{"Error": "Invalid username or password: CODE 001"}
		data["CSRFToken"] = c.Get("csrf").(string)
		return h.tmpl.ExecuteTemplate(c.Response(), "login.go.tmpl", data)
	}
	userGroups := h.getLDAPGroups(user)
	if len(userGroups) == 0 {
		data := map[string]any{"Error": "Invalid username or password: CODE 002"}
		data["CSRFToken"] = c.Get("csrf").(string)
		return h.tmpl.ExecuteTemplate(c.Response(), "login.go.tmpl", data)
	}
	if !h.authService.HasAnyRequiredGroup(userGroups, requireGroups) {
		data := map[string]any{"Error": "Invalid username or password: CODE 003"}
		data["CSRFToken"] = c.Get("csrf").(string)
		return h.tmpl.ExecuteTemplate(c.Response(), "login.go.tmpl", data)
	}
	err = h.authService.GenTokensAndSave(strings.ToLower(username), userGroups, c, ip, ua)
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, "/csplatform/home")
}

func (h *AuthHandler) PostLogout(c echo.Context) error {
	sessionCookie, err := c.Cookie(h.authService.SessionCookie)
	if err != nil {
		return err
	}
	sessionID := sessionCookie.Value
	ctx := context.Background()
	username, err := h.authService.SessionDriver.LookupUserBySID(ctx, sessionID)
	if err != nil {
		return err
	}
	if err := h.authService.SessionDriver.RevokeByUserIDAndSessionID(ctx, username, sessionID); err != nil {
		return err
	}
	security.ExpireSessionCookie(c, h.withTLs, h.authService.SessionCookie)
	registrySessId := getSessionID(c)
	h.codeServerSessionRegistry.CancellAll(registrySessId, false)
	return c.Redirect(http.StatusFound, "/auth/login")
}

func (h *AuthHandler) getLDAPGroups(user auth.Info) []string {
	userGroups := make([]string, 0)
	userExt := user.GetExtensions()
	memberOf := userExt.Values("memberOf")

	if len(memberOf) > 0 {
		userGroups = security.ParseMemberOf(memberOf, "usergroup", h.log)
	}
	return userGroups
}
