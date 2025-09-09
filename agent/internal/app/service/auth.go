package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"a0/internal/app/security"
	"a0/internal/app/xerror"
	"a0/internal/app/xsession"
)

const (
	ACCESS_TOKEN_TTL  = time.Minute * 15
	REFRESH_TOKEN_TTL = time.Hour * 24 * 7
)

type AuthService struct {
	jwtService    *security.JWTService
	withTLS       bool
	log           zerolog.Logger
	SessionDriver *xsession.RedisSessionManager
}

func NewAuthService(
	jwtService *security.JWTService,
	withTLS bool,
	log zerolog.Logger,
	sm *xsession.RedisSessionManager,
) *AuthService {
	return &AuthService{jwtService, withTLS, log, sm}
}

func (s *AuthService) IsLoggedIn(c echo.Context) (string, string, []string, bool, error) {

	ip := c.RealIP()
	ua := c.Request().Header.Get("User-Agent")
	ctx := context.Background()

	sessionID := c.Request().Header.Get("X-Session-ID")
	if sessionID == "" {
		return "", "", []string{}, false, errors.New("missing credentials")
	}

	userID, err := s.SessionDriver.LookupUserBySID(ctx, sessionID)
	if err != nil {
		return "", "", []string{}, false, err
	}

	accessTokenFound := true
	accessToken, err := s.SessionDriver.GetAccessBySID(ctx, userID, sessionID)
	if err != nil {
		if err.Error() == "not found" {
			accessTokenFound = false
		} else {
			return "", "", []string{}, false, err
		}
	}

	if accessTokenFound {
		if uname, groups, err := s.jwtService.JWTValidateAccessToken(accessToken, ip, ua); err == nil && uname != "" {
			// s.log.Debug().Msgf("Access token validated for user: %s", uname)
			return uname, sessionID, groups, true, nil
		} else if !errors.Is(err, jwt.ErrTokenExpired) {
			s.log.Warn().Msgf("Possible Theft access token detected for session: %s %v", sessionID, err)
			return "", "", []string{}, false, &xerror.ErrJWTAccessTokenValidationError{}
		} else {
			s.log.Debug().Msgf("Access token validation err: %s, %v, %v", uname, groups, err)
		}
	}

	refreshToken, err := s.SessionDriver.GetRefreshBySID(ctx, userID, sessionID)
	if err != nil {
		if err.Error() == "not found" {
			return "", "", []string{}, false, &xerror.ErrJWTRefreshTokenNotFound{}
		} else {
			return "", "", []string{}, false, err
		}
	}

	uname, groups, verr := s.jwtService.JWTValidateRefreshToken(refreshToken, ip, ua)
	if verr != nil {
		if errors.Is(verr, jwt.ErrTokenExpired) {
			return "", "", []string{}, false, &xerror.ErrJWTRefreshTokenExpired{}
		}
		s.log.Warn().Msgf("Possible Theft refresh token detected for session: %s", sessionID)
		return "", "", []string{}, false, &xerror.ErrJWTRefreshTokenValidationError{}
	}

	newAccessToken, err := s.jwtService.JWTCreateAccessToken(uname, groups, ip, ua)
	if err != nil {
		return "", "", []string{}, false, err
	}
	newRefreshToken, err := s.jwtService.JWTCreateRefreshToken(uname, groups, ip, ua)
	if err != nil {
		return "", "", []string{}, false, err
	}

	err = s.SessionDriver.RotateOnRefresh(
		ctx,
		userID,
		sessionID,
		refreshToken,
		newAccessToken,
		ACCESS_TOKEN_TTL,
		newRefreshToken,
		REFRESH_TOKEN_TTL,
	)
	if err != nil {
		return "", "", []string{}, false, err
	}
	return uname, sessionID, groups, true, nil
}

func (s *AuthService) HasAnyRequiredGroup(userGroups, required []string) bool {
	if len(required) == 0 {
		return true
	}
	userSet := make(map[string]struct{}, len(userGroups))
	for _, g := range userGroups {
		k := strings.ToLower(strings.TrimSpace(g))
		if k != "" {
			userSet[k] = struct{}{}
		}
	}
	for _, r := range required {
		k := strings.ToLower(strings.TrimSpace(r))
		if _, ok := userSet[k]; ok {
			return true
		}
	}
	return false
}

func (s *AuthService) HasAllRequiredGroups(userGroups, required []string) bool {
	userSet := make(map[string]struct{}, len(userGroups))
	for _, g := range userGroups {
		userSet[strings.ToLower(strings.TrimSpace(g))] = struct{}{}
	}
	for _, r := range required {
		if _, ok := userSet[strings.ToLower(strings.TrimSpace(r))]; !ok {
			return false
		}
	}
	return true
}
