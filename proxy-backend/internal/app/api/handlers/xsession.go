package handlers

import (
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"

	"v0/internal/app/xsession"
)

type CodeServerSessionHandler struct {
	r    *xsession.CodeServerSessionRegistry
	tmpl *template.Template
}

func NewCodeServerSessionHandler(r *xsession.CodeServerSessionRegistry, tmpl *template.Template) *CodeServerSessionHandler {
	return &CodeServerSessionHandler{r, tmpl}
}

func (h *CodeServerSessionHandler) ListSessions(c echo.Context) error {
	return c.JSON(http.StatusOK, h.r.ListSessions())
}

func (h *CodeServerSessionHandler) ListConnections(c echo.Context) error {
	sid := c.Param("sessionID")
	if sid == "" {
		return c.JSON(400, echo.Map{"error": "missing sessionID"})
	}
	if strings.Contains(sid, "%") {
		dec, err := url.PathUnescape(sid)
		if err != nil {
			return c.JSON(400, echo.Map{"error": "Invalid sessionID encoding"})
		}
		sid = dec
	}
	conns := h.r.ListConns(sid)
	if len(conns) == 0 {

	}
	return c.JSON(http.StatusOK, echo.Map{"sessionId": sid, "conns": conns})
}

func (h *CodeServerSessionHandler) ListConnectionsPost(c echo.Context) error {
	var in struct {
		SessionID string `json:"sessionId" form:"sessionId"`
	}
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid body"})
	}
	sid := in.SessionID
	if sid == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "missing sessionId"})
	}
	conns := h.r.ListConns(sid)
	return c.JSON(http.StatusOK, echo.Map{"sessionId": sid, "conns": conns})
}

func (h *CodeServerSessionHandler) CancelConn(c echo.Context) error {
	sid := c.Param("sessionID")
	cid := c.Param("connId")
	ok := h.r.CancelConn(sid, cid)
	if !ok {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
	}
	return c.JSON(http.StatusOK, echo.Map{
		"sessionId": sid,
		"connId":    cid,
		"cancelled": true,
	})
}

func (h *CodeServerSessionHandler) CancelConnPost(c echo.Context) error {
	var in struct {
		SessionID string `json:"sessionId" form:"sessionId"`
		ConnID    string `json:"connId" form:"connId"`
	}
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid body"})
	}
	sid := in.SessionID
	cid := in.ConnID
	ok := h.r.CancelConn(sid, cid)
	if !ok {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
	}
	return c.JSON(http.StatusOK, echo.Map{
		"sessionId": sid,
		"connId":    cid,
		"cancelled": true,
	})
}

func (h *CodeServerSessionHandler) CancelAll(c echo.Context) error {
	sid := c.Param("sessionID")
	n := h.r.CancellAll(sid, true)
	if n == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "not found or already empty"})
	}
	return c.JSON(http.StatusOK, echo.Map{"sessionId": sid, "cancelledCnt": n})
}

func (h *CodeServerSessionHandler) CancelAllPost(c echo.Context) error {
	var in struct {
		SessionID string `json:"sessionId" form:"sessionId"`
	}
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid body"})
	}
	sid := in.SessionID
	n := h.r.CancellAll(sid, true)
	if n == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "not found or already empty"})
	}
	return c.JSON(http.StatusOK, echo.Map{"sessionId": sid, "cancelledCnt": n})
}

func (h *CodeServerSessionHandler) CloseIdle(c echo.Context) error {
	sid := c.Param("sessionID")
	ok := h.r.CloseIdle(sid)
	if !ok {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "no transports for session"})
	}
	return c.JSON(http.StatusOK, echo.Map{"sessionId": sid, "closed": "idle connections closed"})
}

func (h *CodeServerSessionHandler) CloseIdlePost(c echo.Context) error {
	var in struct {
		SessionID string `json:"sessionId" form:"sessionId"`
	}
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid body"})
	}
	sid := in.SessionID
	ok := h.r.CloseIdle(sid)
	if !ok {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "no transports for session"})
	}
	return c.JSON(http.StatusOK, echo.Map{"sessionId": sid, "closed": "idle connections closed"})
}

func (h *CodeServerSessionHandler) RenderPage(c echo.Context) error {
	data := make(map[string]any)
	data["Title"] = "Code Server Session MAnager"
	return h.tmpl.ExecuteTemplate(c.Response(), "cs-session.mng.go.tmpl", data)
}
