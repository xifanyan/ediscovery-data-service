package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/xifanyan/ediscovery-data-service/config"
)

type UserInfo struct {
	Name  string
	Roles map[string]struct{}
}

func UserAuthMiddleware(cfg config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// expecting the user header to be in the format "username:role1,role2,role3"
			userHeader := c.Request().Header.Get("USER")
			log.Debug().Msgf("User Info: %s", userHeader)

			// Trim whitespace and check if header is empty
			userHeader = strings.TrimSpace(userHeader)
			if userHeader == "" {
				return echo.NewHTTPError(http.StatusBadRequest, "USER header is required")
			}

			userInfo, err := parseUserHeader(userHeader)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err)
			}

			if !isCaseManager(cfg, userInfo) {
				msg := fmt.Sprintf("User %s does not have CaseManager role", userInfo.Name)
				return echo.NewHTTPError(http.StatusForbidden, msg)
			}

			// Set the parsed username in the context for potential use in subsequent handlers
			c.Set("user", userInfo.Name)
			return next(c)
		}
	}
}

func isCaseManager(cfg config.Config, userInfo UserInfo) bool {
	for role := range userInfo.Roles {
		m := cfg.RoleMap["CaseManager"]
		log.Debug().Msgf("RoleMap: %+v, role: %s", m, role)
		if _, ok := m[role]; ok {
			return true
		}
	}
	return false
}

func parseUserHeader(header string) (UserInfo, error) {
	var userInfo UserInfo = UserInfo{}

	header = strings.TrimSpace(header)

	parts := strings.SplitN(header, ":", 2)
	if len(parts) != 2 {
		return userInfo, fmt.Errorf("header must be in the format 'username:role1,role2,...'")
	}

	username := strings.TrimSpace(parts[0])
	if username == "" {
		return userInfo, fmt.Errorf("username cannot be empty")
	}

	roles := strings.Split(parts[1], ",")
	roles = filterEmptyStrings(roles)
	if len(roles) == 0 {
		return userInfo, fmt.Errorf("at least one role is required")
	}

	log.Debug().Msgf("username: %s, roles: %v", username, roles)

	return UserInfo{
		Name:  username,
		Roles: roleSliceToMap(roles),
	}, nil
}

func filterEmptyStrings(strs []string) []string {
	var res []string
	for _, s := range strs {
		if s != "" {
			res = append(res, s)
		}
	}
	return res
}

func roleSliceToMap(roles []string) map[string]struct{} {
	rolesMap := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		rolesMap[role] = struct{}{}
	}
	return rolesMap
}
